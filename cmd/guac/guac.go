package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	clientConfig "github.com/appaegis/golang-common/pkg/config"
	"github.com/appaegis/golang-common/pkg/db_data/adaptor"
	"github.com/appaegis/golang-common/pkg/monitorpolicy"
	"github.com/appaegis/golang-common/pkg/storage"
	"github.com/appaegis/golang-common/pkg/utils"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/geoip"
	"github.com/wwt/guac/lib/logging"
	guac "github.com/wwt/guac/pkg"
	guacSession "github.com/wwt/guac/pkg/session"
)

var commitID string

func main() {
	geoip.Init()
	logging.Init()
	defer logging.Close()
	guac.InitK8S()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debugln("Debug level enabled")
	logrus.Traceln("Trace level enabled")

	_ = os.MkdirAll("/efs/rdp", 0o777)
	_ = os.Chmod("/efs/rdp", os.ModePerm)

	go cleanExpiredRdpFiles()

	// XXX
	pmHost := clientConfig.GetPolicyManagementEndPoint()

	servlet := &guac.GuacServerWrapper{Server: guac.NewServer(DemoDoConnect)}
	wsServer := guac.NewWebsocketServer(DemoDoConnect)

	chManagement := guac.NewChannelManagement()

	go connectToAstraea(pmHost, chManagement)

	sessions := guac.NewMemorySessionStore()
	wsServer.OnConnect = sessions.Add
	wsServer.OnDisconnect = sessions.Delete
	wsServer.AppendChannelManagement(chManagement)

	mux := http.NewServeMux()
	mux.Handle("/tunnel", servlet)
	mux.Handle("/tunnel/", servlet)
	mux.Handle("/websocket-tunnel", wsServer)
	mux.HandleFunc("/sessions/", guac.WithMetrics(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		sessions.RLock()
		defer sessions.RUnlock()

		type ConnIds struct {
			Uuid string `json:"uuid"`
			Num  int    `json:"num"`
		}

		connIds := make([]*ConnIds, len(sessions.ConnIds))

		i := 0
		for id, num := range sessions.ConnIds {
			connIds[i] = &ConnIds{
				Uuid: id,
				Num:  num,
			}
		}

		if err := json.NewEncoder(w).Encode(connIds); err != nil {
			logrus.Error(err)
		}
	}))
	mux.Handle("/metrics", promhttp.Handler())

	logrus.Println("Serving on :4567")
	logrus.Println("commit id: " + commitID)

	s := &http.Server{
		Addr:           "0.0.0.0:4567",
		Handler:        mux,
		ReadTimeout:    guac.SocketTimeout,
		WriteTimeout:   guac.SocketTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	err := s.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}

// DemoDoConnect creates the tunnel to the remote machine (via guacd)
func DemoDoConnect(request *http.Request) (guac.Tunnel, error) {
	config := guac.NewGuacamoleConfiguration()

	var query url.Values
	if request.URL.RawQuery == "connect" {
		// http tunnel uses the body to pass parameters
		data, err := ioutil.ReadAll(request.Body)
		if err != nil {
			logrus.Error("Failed to read body ", err)
			return nil, err
		}
		_ = request.Body.Close()
		queryString := string(data)
		query, err = url.ParseQuery(queryString)
		if err != nil {
			logrus.Error("Failed to parse body query ", err)
			return nil, err
		}
		logrus.Debugln("body:", queryString, query)
	} else {
		query = request.URL.Query()
	}

	config.Protocol = query.Get("scheme")
	config.Parameters = map[string]string{}
	for k, v := range query {
		config.Parameters[k] = v[0]
	}
	// no need to pass alert rules specific data to guacmole
	delete(config.Parameters, "tenantId")
	delete(config.Parameters, "alertRules")
	delete(config.Parameters, "clientIp")
	delete(config.Parameters, "role_ids")

	appauthz, err := request.Cookie("appauthz")
	idtoken := ""
	if err == nil {
		idtoken = appauthz.Value
		config.Parameters["gateway-password"] = idtoken
	} else {
		logrus.Errorf("appauthz cookie not found")
		// return nil, fmt.Errorf("appauthz cookie not found")
	}

	tenantId := query.Get("tenantId")
	roleIds := query.Get("roleIds")
	appId := query.Get("appId")
	userId := query.Get("userId")
	userName := query.Get("username")
	appName := query.Get("appName")
	var permissions string
	cli := adaptor.GetDefaultDaoClient()
	if actions := cli.QueryPolicyByAstraea(appId, userId).Actions; actions != nil {
		permissions = strings.Join(actions, ",")
	}
	if !strings.Contains(permissions, "copy") {
		config.Parameters["disable-copy"] = "true"
	}
	if !strings.Contains(permissions, "paste") {
		config.Parameters["disable-paste"] = "true"
	}
	app := cli.QueryResource(appId)
	sku := cli.GetTenantById(tenantId).TenantType
	clientIp := strings.Split(query.Get("clientIp"), ":")[0]
	clientPrivateIp := query.Get("clientPrivateIp")
	logrus.Infof("app %s, user %s, permissions %s, ip %s, private ip %s, recording %v", appId, userId, permissions, clientIp, clientPrivateIp, app != nil && app.EnableRecording)

	// session recording
	sessionId := uuid.NewV4()
	s3key := time.Now().Format(time.RFC3339)
	enableRecording := false
	sessionDataKey := sessionId.String()

	loggingInfo := logging.NewLoggingInfo(tenantId, userId, appName, clientIp, s3key, sku, enableRecording, clientPrivateIp)
	if app != nil && app.EnableRecording {
		loggingInfo.EnableRecording = true
		config.Parameters["recording-path"] = "/efs/rdp"
		config.Parameters["create-recording-path"] = "true"
		config.Parameters["recording-include-keys"] = "true"
		config.Parameters["recording-name"] = loggingInfo.GetRecordingFileName()
	}

	shareSessionID := query.Get("shareSessionId")
	if room, ok := guac.GetRoomByAppIdAndCreator(appId, userId); ok {
		logrus.Infof("host user %s join to existing session %s, app %s", userId, room.SessionId, room.AppId)
		shareSessionID = room.SessionId
	}
	session := &guacSession.SessionCommonData{}
	if shareSessionID == "" { // launch a new rdp session
		logrus.Infof("sessionId %s", sessionDataKey)
		session.TenantID = tenantId
		session.AppID = appId
		session.Email = userId
		session.UserName = userName
		session.RoleIDs = strings.Split(roleIds, ",")
		session.IDToken = idtoken
		session.ClientIsoCountry = geoip.GetIpIsoCode(query.Get("clientIp"))
		session.ClientIP = clientIp
		session.ClientPrivateIp = clientPrivateIp
		session.SessionStartTime = time.Now()
		session.AppName = appName
		session.RdpSessionId = sessionDataKey

		if app.MonitorPolicyEntryId != "" {
			monitorPolicy := cli.QueryMonitorPolicyEntryById(app.MonitorPolicyEntryId)
			session.MonitorPolicyId = monitorPolicy.ID
			session.MonitorPolicyName = monitorPolicy.Name
			session.MonitorRules = monitorpolicy.QueryMonitorRuleForUser(monitorPolicy.ID, userId)
		}
		_, fail := storage.GetStorageByTenantId(tenantId, clientConfig.GetRegion())
		if app.EnableRecording && !fail {
			session.Recording = true
		}

		guac.SessionDataStore.Set(sessionDataKey, session)
	} else { // join a existing rdp session
		sessionData := guac.SessionDataStore.Get(shareSessionID)
		room, ok := guac.GetRdpSessionRoom(shareSessionID)
		if sessionData == nil || !ok {
			return nil, fmt.Errorf("session not found by session id %s", shareSessionID)
		}
		config.ConnectionID = room.RdpConnectionId
		session = sessionData.(*guacSession.SessionCommonData)
		loggingInfo.AppName = session.AppName
	}

	if query.Get("width") != "" {
		config.OptimalScreenHeight, err = strconv.Atoi(query.Get("width"))
		if err != nil || config.OptimalScreenHeight == 0 {
			logrus.Error("Invalid height")
			config.OptimalScreenHeight = 600
		}
	}
	if query.Get("height") != "" {
		config.OptimalScreenWidth, err = strconv.Atoi(query.Get("height"))
		if err != nil || config.OptimalScreenWidth == 0 {
			logrus.Error("Invalid width")
			config.OptimalScreenWidth = 800
		}
	}
	config.AudioMimetypes = []string{"audio/L16", "rate=44100", "channels=2"}

	var conn net.Conn
	if shareSessionID != "" {
		logrus.Infof("Connecting to guacd %s", session.GuacdAddr)
		conn, err = net.Dial("tcp", session.GuacdAddr)
		if err != nil {
			logrus.Errorf("err connecting to guacd %s %v", session.GuacdAddr, err)
			return nil, err
		}
	} else {
		addr := "127.0.0.1:4822"
		if os.Getenv("POD_IP") != "" {
			addr, err = guac.GetGuacdTarget()
			if err != nil {
				return nil, err
			}
			addr = addr + ":4822"
		}
		session.GuacdAddr = addr
		logrus.Infof("Connecting to guacd %s", session.GuacdAddr)

		conn, err = net.Dial("tcp", addr)
		if err != nil {
			logrus.Errorln("error while connecting to guacd", err)
			return nil, err
		}
	}

	stream := guac.NewStream(conn, guac.SocketTimeout)

	// logrus.Debugf("Starting handshake with %#v", config)
	err = stream.Handshake(config)
	if err != nil {
		logrus.Infof("handshake failed: %v %T", err, err)
		return nil, err
	}
	return guac.NewSimpleTunnel(stream, sessionId, loggingInfo), nil
}

func connectToAstraea(pmHost string, chManagement *guac.ChannelManagement) {
	for {
		url := fmt.Sprintf("ws://%s/ws", pmHost)
		logrus.Infof("ws connecting to %s", url)

		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			logrus.Fatalf("dial: %s", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		for {
			request := PolicyNotifyRequest{}
			err = c.ReadJSON(&request)
			if err != nil {
				logrus.Errorf("read from ws err: %s", err.Error())
				c.Close()
				break
			} else {
				//  logrus.Infof("received msg %#v", request)
				for _, event := range request.Events {
					for _, id := range event.IDs {
						_ = chManagement.BroadCast(id, 1)
					}
				}
			}
		}
	}
}

func cleanExpiredRdpFiles() {
	tick := time.NewTicker(10 * time.Minute)
	defer tick.Stop()
	for range tick.C {
		utils.CleanExpiredFiles("/efs/rdp/*", "*", 24*time.Hour)
	}
}

type PolicyNotifyEvent struct {
	TypeName string   `json:"typeName"`
	IDs      []string `json:"ids"`
}
type PolicyNotifyRequest struct {
	Events []PolicyNotifyEvent `json:"events"`
}
