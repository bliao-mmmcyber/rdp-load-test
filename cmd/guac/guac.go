package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/wwt/guac"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debugln("Debug level enabled")
	logrus.Traceln("Trace level enabled")

	// XXX
	etcdCli := guac.NewEtcdClient()
	pmRes := guac.EtcdGet(etcdCli, "/dplocal/dp_setting/POLICY_MANAGEMENT_ENDPOINT")
	pmHost := string(pmRes.Kvs[0].Value)
	selfRes := guac.EtcdGet(etcdCli, "/dplocal/dp_setting/RDPWS_HOST")
	selfHost := string(selfRes.Kvs[0].Value)
	requestBody := map[string]string{}
	requestBody["address"] = selfHost
	requestBytes, _err := json.Marshal(requestBody)
	if _err != nil {
		logrus.Fatal("marshal failed")
		return
	}
	resp, _err := http.Post(fmt.Sprintf("http://%s/register", pmHost), "application/json", bytes.NewBuffer(requestBytes))
	if _err != nil {
		logrus.Fatal("marshal failed")
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	logrus.Println("response Body:", string(body))
	// XXX

	servlet := guac.NewServer(DemoDoConnect)
	wsServer := guac.NewWebsocketServer(DemoDoConnect)

	chManagement := guac.NewChannelManagement()
	chManagement.RequestPolicyFunc = func(appID string, userID string) []string {
		requestParam := url.Values{
			"userID": []string{userID},
			"appID":  []string{appID},
		}
		resp, _err := http.Get(fmt.Sprintf("http://%s/policy?%s", pmHost, requestParam.Encode()))
		if _err != nil {
			logrus.Fatal("marshal failed")
			return nil
		}
		defer resp.Body.Close()

		var p PolicyResponse
		body, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(body, &p)
		if p.Actions != nil {
			return p.Actions
		}
		return nil
	}

	sessions := guac.NewMemorySessionStore()
	wsServer.OnConnect = sessions.Add
	wsServer.OnDisconnect = sessions.Delete
	wsServer.AppendChannelManagement(chManagement)

	mux := http.NewServeMux()
	mux.Handle("/tunnel", servlet)
	mux.Handle("/tunnel/", servlet)
	mux.Handle("/websocket-tunnel", wsServer)
	mux.HandleFunc("/policy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var p PolicyNotifyRequest
			err := json.NewDecoder(r.Body).Decode(&p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for _, event := range p.Events {
				for _, id := range event.IDs {
					chManagement.BroadCast(id, 1)
				}
			}
		} else {
			http.Error(w, fmt.Sprint("not allow method"), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/sessions/", func(w http.ResponseWriter, r *http.Request) {
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
	})

	logrus.Println("Serving on :4567")

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

	var err error
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
		addr, err = net.ResolveTCPAddr("tcp", "guacd-service:4822")
	} else {
		addr, err = net.ResolveTCPAddr("tcp", "127.0.0.1:4822")
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
	return guac.NewSimpleTunnel(stream), nil
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
