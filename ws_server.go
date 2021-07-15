package guac

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"strconv"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
)

var appaegisCmdOpcodeIns = []byte("5.AACMD")
var downloadCmdOpcodeIns = []byte("3.get")
var uploadCmdOpcodeIns = []byte("3.put")

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

	logrus.Debug("Connecting to tunnel")
	var tunnel Tunnel
	var e error
	if s.connect != nil {
		tunnel, e = s.connect(r)
	} else {
		tunnel, e = s.connectWs(ws, r)
	}
	if e != nil {
		return
	}
	defer func() {
		if err = tunnel.Close(); err != nil {
			logrus.Traceln("Error closing tunnel", err)
		}
	}()
	logrus.Debug("Connected to tunnel")

	query := r.URL.Query()
	userId := query.Get("userId")
	appId := query.Get("appId")
	logrus.Debug("Query Parameters userId:", userId)
	logrus.Debug("Query Parameters appId:", appId)

	sessionDataKey := appId + "/" + userId
	defer func() {
		logrus.Infof("session data delete: %s", sessionDataKey)
		SessionDataStore.Delete(sessionDataKey)
	}()
	if data, ok := SessionDataStore.Get(sessionDataKey).(SessionAlertRuleData); ok {
		logrus.Infof("session data alert rules: %v", data)
	}

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

	if s.channelManagement != nil {
		ch := make(chan int, 1)
		channelID := uuid.NewV4()
		defer s.channelManagement.Remove(appId, userId, channelID.String())
		if userId != "" {
			s.channelManagement.Add(userId, channelID.String(), ch)
		}
		if appId != "" {
			s.channelManagement.Add(appId, channelID.String(), ch)
		}

		go BroadCastToWs(ws, ch, appId, userId, s.channelManagement.RequestPolicyFunc)
	}

	go wsToGuacd(ws, writer, sessionDataKey)

	guacdToWs(ws, reader)
}

// MessageReader wraps a websocket connection and only permits Reading
type MessageReader interface {
	// ReadMessage should return a single complete message to send to guac
	ReadMessage() (int, []byte, error)
}

func wsToGuacd(ws *websocket.Conn, guacd io.Writer, sessionDataKey string) {
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

		// download/upload check
		// if bytes.HasPrefix(data, downloadCmdOpcodeIns) ||
		// 	bytes.HasPrefix(data, uploadCmdOpcodeIns) {
		// 	handleAppaegisCommand(ws, data, sessionDataKey)
		// 	continue
		// }

		if bytes.HasPrefix(data, appaegisCmdOpcodeIns) {
			handleAppaegisCommand(ws, data, sessionDataKey)
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
				logrus.Traceln("Failed sending message to ws", err)
				return
			}
			buf.Reset()
		}
	}
}

func BroadCastToWs(ws MessageWriter, ch chan int, appId string, userId string, requestPolicy func(string, string) []string) {
	logrus.Debug("create BroadCastToWs")
	BroadCastPolicy(ws, appId, userId, requestPolicy)
	for op := range ch {
		if op == 1 {
			BroadCastPolicy(ws, appId, userId, requestPolicy)
		}
	}
}
func BroadCastPolicy(ws MessageWriter, appId string, userId string, requestPolicy func(string, string) []string) {
	actions := requestPolicy(appId, userId)
	if actions == nil {
		logrus.Debug("policy empty:", appId, userId)
		return
	}
	instruction := []string{"policy"}
	instruction = append(instruction, actions...)
	ins := NewInstruction("sync", instruction...)
	insValue := ins.String()
	logrus.Debug("send:", insValue)
	if err := ws.WriteMessage(1, []byte(insValue)); err != nil {
		if err == websocket.ErrCloseSent {
			return
		}
		logrus.Traceln("(testToWs) Failed sending message to ws", err)
		return
	}
}

// J json response helper type
type J map[string]interface{}

func handleAppaegisCommand(ws *websocket.Conn, cmd []byte, sessionDataKey string) {
	logrus.Printf("receive: %s\n", cmd)
	instruction, err := Parse(cmd)
	if err != nil {
		logrus.Println("Instruction parse error: ", err)
		return
	}
	ses, ok := SessionDataStore.Get(sessionDataKey).(*SessionAlertRuleData)
	if !ok {
		logrus.Infof("session data not found: %s", sessionDataKey)
		return
	}
	logrus.Infof("session data %v", ses)

	result := J{}
	op := instruction.Args[0]
	requestID := instruction.Args[1]
	logrus.Printf("op: %s, requestID: %s", op, requestID)

	// operations switch
	if op == "download-check" {
		fileCount, err := strconv.Atoi(instruction.Args[2])
		if err != nil {
			fileCount = 1
		}
		ok, block := CheckAlertRule(ses, "download", fileCount)
		if !ok && block {
			result = J{
				"ok": false,
			}
		} else {
			result = J{
				"ok": true,
				"prompt": !ok,
			}
		}
	} else if op == "log-download" {
		fileCount, err := strconv.Atoi(instruction.Args[2])
		if err != nil {
			fileCount = 1
		}
		logging.Log(logging.Action{
			AppTag:    "guac.download",
			TenantID:  ses.TenantID,
			UserEmail: ses.Email,
			AppID:     ses.AppID,
			RoleIDs:   ses.RoleIDs,
			FileCount: fileCount,
			ClientIP:  ses.ClientIP,
		})
		IncrAlertRuleSessionCountByNumber(ses, "download", fileCount)
		result = J{
			"ok": true,
			"count": fileCount,
		}
	} else {
		logging.Log(logging.Action{
			AppTag:    "guac." + strings.ToLower(op),
			TenantID:  ses.TenantID,
			UserEmail: ses.Email,
			AppID:     ses.AppID,
			RoleIDs:   ses.RoleIDs,
			ClientIP:  ses.ClientIP,
		})
		result = J{
			"ng": 1,
		}
	}

	resultJSON, _ := json.Marshal(result)
	respInstruction := NewInstruction("appaegis-resp", requestID, string(resultJSON))
	resp := []byte(respInstruction.String())
	logrus.Debug("appaegis cmd send: ", string(resp))
	if err := ws.WriteMessage(websocket.TextMessage, resp); err != nil {
		logrus.Println("write error: ", err)
	}
}
