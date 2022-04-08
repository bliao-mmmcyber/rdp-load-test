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

	"github.com/appaegis/golang-common/pkg/config"
	"github.com/appaegis/golang-common/pkg/dynamodbcli"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/geoip"
	"github.com/wwt/guac/lib/logging"
	guac "github.com/wwt/guac/pkg"
)

var commitID string

func main() {
	geoip.Init()
	logging.Init()
	defer logging.Close()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debugln("Debug level enabled")
	logrus.Traceln("Trace level enabled")

	_ = os.MkdirAll("/efs/rdp", 0o777)
	_ = os.Chmod("/efs/rdp", os.ModePerm)

	// XXX
	pmHost := config.GetPolicyManagementEndPoint()

	servlet := &guac.GuacServerWrapper{Server: guac.NewServer(DemoDoConnect)}
	wsServer := guac.NewWebsocketServer(DemoDoConnect)

	chManagement := guac.NewChannelManagement()
	chManagement.RequestPolicyFunc = requestPolicy

	go connectToAstraea(pmHost, chManagement)

	sessions := guac.NewMemorySessionStore()
	wsServer.OnConnect = sessions.Add
	wsServer.OnDisconnect = sessions.Delete
	wsServer.AppendChannelManagement(chManagement)

	mux := http.NewServeMux()
	mux.Handle("/tunnel", servlet)
	mux.Handle("/tunnel/", servlet)
	mux.Handle("/websocket-tunnel", wsServer)
	mux.HandleFunc("/policy", guac.WithMetrics(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var p PolicyNotifyRequest
			err := json.NewDecoder(r.Body).Decode(&p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for _, event := range p.Events {
				for _, id := range event.IDs {
					_ = chManagement.BroadCast(id, 1)
				}
			}
		} else {
			http.Error(w, "not allow method", http.StatusInternalServerError)
		}
	}))
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

func requestPolicy(appID string, userID string) []string {
	requestParam := url.Values{
		"userID": []string{userID},
		"appID":  []string{appID},
	}
	resp, _err := http.Get(fmt.Sprintf("http://%s/policy?%s", config.GetPolicyManagementEndPoint(), requestParam.Encode()))
	if _err != nil {
		logrus.Fatalf("get policy failed, %s", _err.Error())
		return nil
	}
	defer resp.Body.Close()

	var p PolicyResponse
	body, _ := ioutil.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &p)
	if p.Actions != nil {
		return p.Actions
	}
	return nil
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
	if err == nil {
		config.Parameters["gateway-password"] = appauthz.Value
	}

	// AC-938: alert rules
	tenantId := query.Get("tenantId")
	roleIds := query.Get("roleIds")
	appId := query.Get("appId")
	userId := query.Get("userId")
	appName := query.Get("appName")
	var permissions string
	if actions := requestPolicy(appId, userId); actions != nil {
		permissions = strings.Join(actions, ",")
	}
	logrus.Infof("app %s, user %s, permissions %s", appId, userId, permissions)
	if !strings.Contains(permissions, "copy") {
		config.Parameters["disable-copy"] = "true"
	}
	if !strings.Contains(permissions, "paste") {
		config.Parameters["disable-paste"] = "true"
	}

	app := dynamodbcli.GetResourceById(appId)

	// session recording
	streamId := uuid.NewV4()
	s3key := time.Now().Format(time.RFC3339)
	sku := dynamodbcli.GetTenantById(tenantId).TenantType
	clientIp := strings.Split(query.Get("clientIp"), ":")[0]
	loggingInfo := logging.NewLoggingInfo(tenantId, userId, appName, clientIp, s3key, sku, app.EnableRecording)

	if app.EnableRecording {
		config.Parameters["recording-path"] = "/efs/rdp"
		config.Parameters["create-recording-path"] = "true"
		config.Parameters["recording-include-keys"] = "true"
		config.Parameters["recording-name"] = loggingInfo.GetRecordingFileName()
	}

	logging.Log(logging.Action{
		AppTag:    "guac.connect",
		UserEmail: userId,
		AppID:     appId,
		RoleIDs:   strings.Split(roleIds, ","),
		ClientIP:  strings.Split(query.Get("clientIp"), ":")[0],
	})

	alertRulesString := query.Get("alertRules")
	sessionDataKey := appId + "/" + userId

	sessionAlertRuleData := &guac.SessionCommonData{}
	sessionAlertRuleData.TenantID = tenantId
	sessionAlertRuleData.AppID = appId
	sessionAlertRuleData.Email = userId
	sessionAlertRuleData.RoleIDs = strings.Split(roleIds, ",")
	sessionAlertRuleData.IDToken = appauthz.Value
	sessionAlertRuleData.ClientIsoCountry = geoip.GetIpIsoCode(query.Get("clientIp"))
	sessionAlertRuleData.ClientIP = clientIp
	sessionAlertRuleData.SessionStartTime = time.Now().Truncate(time.Minute).Unix() * 1000
	sessionAlertRuleData.AppName = appName
	sessionAlertRuleData.RuleIDs = make(map[string][]string)
	sessionAlertRuleData.Rules = make(map[string]*guac.AlertRuleData)

	alertRules := []guac.AlertRuleData{}
	if err := json.Unmarshal([]byte(alertRulesString), &alertRules); err != nil {
		logrus.Infof("alertRulesString %s", alertRulesString)
		logrus.Errorf("failed to unmarshal alert rules %s", err.Error())
	} else {

		logrus.Printf("role ids: %v", roleIds)
		for i := range alertRules {
			data := alertRules[i]
			sessionAlertRuleData.Rules[data.RuleID] = &data
			for _, action := range data.EventTypes {
				sessionAlertRuleData.RuleIDs[action] = append(sessionAlertRuleData.RuleIDs[action], data.RuleID)
			}
		}
	}
	guac.SessionDataStore.Set(sessionDataKey, sessionAlertRuleData)
	logrus.Printf("session data stored %s %v", sessionDataKey, sessionAlertRuleData)

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

	logrus.Debug("Connecting to guacd")
	var addr *net.TCPAddr
	if os.Getenv("POD_IP") != "" {
		addr, _ = net.ResolveTCPAddr("tcp", "guacd-service:4822")
	} else {
		addr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:4822")
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		logrus.Errorln("error while connecting to guacd", err)
		return nil, err
	}

	stream := guac.NewStream(conn, guac.SocketTimeout)

	logrus.Debug("Connected to guacd")
	if request.URL.Query().Get("uuid") != "" {
		config.ConnectionID = request.URL.Query().Get("uuid")
	}
	logrus.Debugf("Starting handshake with %#v", config)
	err = stream.Handshake(config)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Socket configured")

	return guac.NewSimpleTunnel(stream, streamId, loggingInfo), nil
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
				logrus.Infof("received msg %#v", request)
				for _, event := range request.Events {
					for _, id := range event.IDs {
						_ = chManagement.BroadCast(id, 1)
					}
				}
			}
		}
	}
}

type PolicyNotifyEvent struct {
	TypeName string   `json:"typeName"`
	IDs      []string `json:"ids"`
}
type PolicyNotifyRequest struct {
	Events []PolicyNotifyEvent `json:"events"`
}

type PolicyResponse struct {
	Actions []string `json:"actions"`
}
