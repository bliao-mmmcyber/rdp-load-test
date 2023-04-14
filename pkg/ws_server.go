package guac

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/appaegis/golang-common/pkg/dynamodbcli"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
)

var (
	appaegisCmdOpcodeIns = []byte("5.AACMD")
	keyCmdOpcodeIns      = []byte("3.key")
	mouseCmdOpcodeIns    = []byte("5.mouse")
	sizeCmdOpcodeIns     = []byte("4.size")
)

// WebsocketServer implements a websocket-based connection to guacd.
type WebsocketServer struct {
	connect   func(*http.Request) (Tunnel, error)
	connectWs func(*websocket.Conn, *http.Request) (Tunnel, error)

	// OnConnect is an optional callback called when a websocket connects.
	// Deprecated: use OnConnectWs
	OnConnect func(string, *http.Request)
	// OnDisconnect is an optional callback called when the websocket disconnects.
	// Deprecated: use OnDisconnectWs
	OnDisconnect func(string, *http.Request, Tunnel)

	// OnConnectWs is an optional callback called when a websocket connects.
	OnConnectWs func(string, *websocket.Conn, *http.Request)
	// OnDisconnectWs is an optional callback called when the websocket disconnects.
	OnDisconnectWs func(string, *websocket.Conn, *http.Request, Tunnel)

	channelManagement *ChannelManagement
}

// NewWebsocketServer creates a new server with a simple connect method.
func NewWebsocketServer(connect func(*http.Request) (Tunnel, error)) *WebsocketServer {
	return &WebsocketServer{
		connect: connect,
	}
}

// NewWebsocketServerWs creates a new server with a connect method that takes a websocket.
func NewWebsocketServerWs(connect func(*websocket.Conn, *http.Request) (Tunnel, error)) *WebsocketServer {
	return &WebsocketServer{
		connectWs: connect,
	}
}

const (
	websocketReadBufferSize  = MaxGuacMessage
	websocketWriteBufferSize = MaxGuacMessage * 2
)

func (s *WebsocketServer) AppendChannelManagement(cm *ChannelManagement) {
	s.channelManagement = cm
}

func (s *WebsocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  websocketReadBufferSize,
		WriteBufferSize: websocketWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true // TODO
		},
	}
	protocol := r.Header.Get("Sec-Websocket-Protocol")
	ws, err := upgrader.Upgrade(w, r, http.Header{
		"Sec-Websocket-Protocol": {protocol},
	})
	if err != nil {
		logrus.Error("Failed to upgrade websocket", err)
		return
	}
	defer func() {
		if err = ws.Close(); err != nil {
			logrus.Traceln("Error closing websocket", err)
		}
	}()
	query := r.URL.Query()
	shareSessionId := query.Get("shareSessionId")
	userId := query.Get("userId")
	appId := query.Get("appId")
	userName := query.Get("username")
	host := query.Get("hostname")
	if strings.HasSuffix(host, "appaegis.tunnel") {
		host = strings.SplitN(host, "-", 2)[0]
	}

	var sharePermissions string
	if shareSessionId != "" { // auth check
		valid, permissions := AuthShare(userId, shareSessionId)
		if !valid {
			logrus.Infof("auth share failed, user %s, session %s", userId, shareSessionId)
			return
		}
		sharePermissions = permissions
	} else {
		if r, ok := GetRoomByAppIdAndCreator(appId, userId); ok { // host user re-connect
			shareSessionId = r.SessionId
			sharePermissions = "keyboard,mouse,admin"
		}
	}

	logrus.Debug("Connecting to tunnel")
	var tunnel Tunnel
	var e error
	if s.connect != nil {
		tunnel, e = s.connect(r)
	} else {
		tunnel, e = s.connectWs(ws, r)
	}
	if e != nil {
		logrus.Errorf("connect to rdp failed %v", e)
		return
	}
	defer func() {
		if err = tunnel.Close(); err != nil {
			logrus.Traceln("Error closing tunnel", err)
		}
	}()
	logrus.Debug("Connected to tunnel")

	sessionId := tunnel.GetUUID()
	id := tunnel.ConnectionID()

	if s.OnConnect != nil {
		s.OnConnect(id, r)
	}
	if s.OnConnectWs != nil {
		s.OnConnectWs(id, ws, r)
	}

	writer := tunnel.AcquireWriter()
	reader := tunnel.AcquireReader()

	if s.OnDisconnect != nil {
		defer s.OnDisconnect(id, r, tunnel)
	}
	if s.OnDisconnectWs != nil {
		defer s.OnDisconnectWs(id, ws, r, tunnel)
	}

	defer tunnel.ReleaseWriter()
	defer tunnel.ReleaseReader()

	sharing := false
	if s.channelManagement != nil {
		if userId != "" && appId != "" {
			ch := make(chan int, 1)
			channelID := uuid.NewV4()
			defer func() { _ = s.channelManagement.Remove(appId, userId, channelID.String()) }()
			if userId != "" {
				_ = s.channelManagement.Add(userId, channelID.String(), ch)
			}
			if appId != "" {
				_ = s.channelManagement.Add(appId, channelID.String(), ch)
				app := dynamodbcli.Singleon().QueryResource(appId)
				sharing = app.AllowSharing
			}
			go BroadCastToWs(ws, ch, sharing, appId, userId)
		}
	}

	var client *RdpClient
	ses, _ := SessionDataStore.Get(sessionId).(*SessionCommonData)
	if shareSessionId == "" { // rdp session owner connected
		clientIp := strings.Split(query.Get("clientIp"), ":")[0]
		logging.Log(logging.Action{
			AppTag:            "rdp.open",
			RdpSessionId:      sessionId,
			UserEmail:         userId,
			Username:          userName,
			AppID:             appId,
			AppName:           tunnel.GetLoggingInfo().AppName,
			TenantID:          tunnel.GetLoggingInfo().TenantId,
			RoleIDs:           strings.Split(query.Get("roleIds"), ","),
			ClientIP:          clientIp,
			TargetIp:          host,
			Recording:         ses.Recording,
			PolicyID:          ses.PolicyID,
			PolicyName:        ses.PolicyName,
			MonitorPolicyId:   ses.MonitorPolicyId,
			MonitorPolicyName: ses.MonitorPolicyName,
		})

		e := dbAccess.SaveActiveRdpSession(&dynamodbcli.ActiveRdpSession{
			Id:        sessionId,
			Owner:     userId,
			TenantId:  tunnel.GetLoggingInfo().TenantId,
			CreatedAt: time.Now(),
		})
		if e != nil {
			logrus.Errorf("save active rdp session failed")
		}
		e = kv.PutWithTimeout(fmt.Sprintf("guac-%s", sessionId), GuacIp+":4567", 24*time.Hour)
		if e != nil {
			logrus.Errorf("put to cache failed %v", e)
		}
		client = NewRdpSessionRoom(sessionId, userId, ws, tunnel.ConnectionID(), sharing, appId, tunnel.GetLoggingInfo().AppName, tunnel.GetLoggingInfo())
		if room, ok := GetRdpSessionRoom(sessionId); ok {
			room.ClientIp = clientIp
		}
	} else {
		sessionId = shareSessionId
		ses, _ := SessionDataStore.Get(sessionId).(*SessionCommonData)
		client, e = JoinRoom(sessionId, userId, ws, sharePermissions)
		if e != nil {
			logrus.Errorf("join to room failed %s", sessionId)
			return
		}
		if room, ok := GetRdpSessionRoom(sessionId); ok {
			logging.Log(logging.Action{
				AppTag:            "rdp.join",
				RdpSessionId:      sessionId,
				UserEmail:         userId,
				AppID:             room.AppId,
				TenantID:          room.TenantId,
				ClientIP:          strings.Split(query.Get("clientIp"), ":")[0],
				Recording:         ses.Recording,
				PolicyID:          ses.PolicyID,
				PolicyName:        ses.PolicyName,
				MonitorPolicyId:   ses.MonitorPolicyId,
				MonitorPolicyName: ses.MonitorPolicyName,
			})
		}
	}

	IncRdpCount(tunnel.GetLoggingInfo().TenantId)
	defer DecRdpCount(tunnel.GetLoggingInfo().TenantId)

	client.SendPermission()

	go wsToGuacd(ws, writer, sessionId, client)
	guacdToWs(ws, reader)

	logrus.Infof("%s leave %s, connection id %s", userId, sessionId, tunnel.ConnectionID())
	e = LeaveRoom(sessionId, userId, tunnel.GetLoggingInfo().TenantId, appId)
	if e != nil {
		logrus.Errorf("leave room failed, session %s, e %v", sessionId, e)
	}

	logrus.Info("server HTTP done")
}

// MessageReader wraps a websocket connection and only permits Reading
type MessageReader interface {
	// ReadMessage should return a single complete message to send to guac
	ReadMessage() (int, []byte, error)
}

func wsToGuacd(ws *websocket.Conn, guacd io.Writer, sessionDataKey string, client *RdpClient) {
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			logrus.Traceln("Error reading message from ws", err)
			return
		}

		if bytes.HasPrefix(data, internalOpcodeIns) {
			// messages starting with the InternalDataOpcode are never sent to guacd
			continue
		}

		if bytes.HasPrefix(data, appaegisCmdOpcodeIns) {
			handleAppaegisCommand(client, data, sessionDataKey)
			continue
		}
		if client.Role != ROLE_ADMIN && bytes.HasPrefix(data, sizeCmdOpcodeIns) {
			continue
		}
		if !client.Mouse && bytes.HasPrefix(data, mouseCmdOpcodeIns) {
			continue
		}
		if !client.Keyboard && bytes.HasPrefix(data, keyCmdOpcodeIns) {
			continue
		}

		if _, err = guacd.Write(data); err != nil {
			logrus.Traceln("Failed writing to guacd", err)
			return
		}
	}
}

// MessageWriter wraps a websocket connection and only permits Writing
type MessageWriter interface {
	// WriteMessage writes one or more complete guac commands to the websocket
	WriteMessage(int, []byte) error
}

func guacdToWs(ws MessageWriter, guacd InstructionReader) {
	buf := bytes.NewBuffer(make([]byte, 0, MaxGuacMessage*2))

	for {
		ins, err := guacd.ReadSome()
		if err != nil {
			logrus.Traceln("Error reading from guacd", err)
			return
		}

		if bytes.HasPrefix(ins, internalOpcodeIns) {
			// messages starting with the InternalDataOpcode are never sent to the websocket
			continue
		}

		if _, err = buf.Write(ins); err != nil {
			logrus.Traceln("Failed to buffer guacd to ws", err)
			return
		}

		// if the buffer has more data in it or we've reached the max buffer size, send the data and reset
		if !guacd.Available() || buf.Len() >= MaxGuacMessage {
			bufbytes := buf.Bytes()
			// bufString := string(bufbytes)
			// logrus.Debug("got buffer:", bufString)
			if err = ws.WriteMessage(1, bufbytes); err != nil {
				if err == websocket.ErrCloseSent {
					return
				}
				logrus.Errorf("Failed sending message to ws %v", err)
				return
			}
			buf.Reset()
		}
	}
}

func BroadCastToWs(ws MessageWriter, ch chan int, sharing bool, appId string, userId string) {
	logrus.Debug("create BroadCastToWs")
	BroadCastPolicy(ws, sharing, appId, userId)
	for op := range ch {
		if op == 1 {
			BroadCastPolicy(ws, sharing, appId, userId)
		}
	}
}

func BroadCastPolicy(ws MessageWriter, sharing bool, appId string, userId string) {
	actions := dynamodbcli.QueryPolicyByAstraea(appId, userId).Actions
	if actions == nil {
		logrus.Debug("policy empty:", appId, userId)
		return
	}
	instruction := []string{"policy"}
	instruction = append(instruction, actions...)
	logrus.Debugf("sharing %v", sharing)
	if sharing {
		instruction = append(instruction, "share")
	}
	ins := NewInstruction("sync", instruction...)
	insValue := ins.String()
	logrus.Debug("send:", insValue)
	if err := ws.WriteMessage(1, []byte(insValue)); err != nil {
		logrus.Error("Failed sending policy message to ws", err)
		if err == websocket.ErrCloseSent {
			return
		}
		return
	}
}

func GetDrivePathInEFS(tenantID, appID, userID string) string {
	return fmt.Sprintf("/efs/rdp/rdp_system_%s_%s_%s", tenantID, appID, userID)
}

// J json response helper type
type J map[string]interface{}

func handleAppaegisCommand(client *RdpClient, cmd []byte, sessionDataKey string) {
	logrus.Printf("receive: %s\n", cmd)
	instruction, err := Parse(cmd)
	if err != nil {
		logrus.Println("Instruction parse error: ", err)
		return
	}
	ses, ok := SessionDataStore.Get(sessionDataKey).(*SessionCommonData)
	if !ok {
		logrus.Infof("session data not found: %s", sessionDataKey)
		return
	}

	// result := J{} //nolint
	var result *Instruction
	op := instruction.Args[1]
	requestID := instruction.Args[0]
	command, e := GetCommandByOp(instruction)
	if e == nil {
		result = command.Exec(instruction, ses, client)
	} else {
		logging.Log(logging.Action{
			AppTag:       "guac." + strings.ToLower(op),
			TenantID:     ses.TenantID,
			UserEmail:    ses.Email,
			AppID:        ses.AppID,
			RoleIDs:      ses.RoleIDs,
			ClientIP:     ses.ClientIP,
			RdpSessionId: ses.RdpSessionId,
		})
		j := J{
			"ng": 1,
		}
		data, _ := json.Marshal(j)
		result = NewInstruction(APPAEGIS_RESP_OP, requestID, string(data))
	}
	if result != nil {
		client.WriteMessage(result)
	}
}
