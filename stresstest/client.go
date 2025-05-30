package stresstest

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	guac "github.com/wwt/guac/pkg"
)

var (
	APP_ID     = "fbd380f9-2e2c-4c22-a8d3-7b9e9632727d"
	NETWORK_ID = "ab528cb3-9b74-425f-896f-a723fffc7a2a"
	SEM        = "qa.sem.mammothcyber.io"
	TENANT_ID  = "583ed688-ae8d-4e85-a6ed-669c328108a4"
	CE         = "qa.ce.mammothcyber.io"
)

func init() {
	if os.Getenv("APP_ID") != "" {
		APP_ID = os.Getenv("APP_ID")
	}
	if os.Getenv("NETWORK_ID") != "" {
		NETWORK_ID = os.Getenv("NETWORK_ID")
	}
	if os.Getenv("SEM") != "" {
		SEM = os.Getenv("SEM")
	}
	if os.Getenv("TENANT_ID") != "" {
		TENANT_ID = os.Getenv("TENANT_ID")
	}
	if os.Getenv("CE") != "" {
		CE = os.Getenv("CE")
	}
	logrus.Infof("app %s, network %s, sem %s, tenant %s, ce %s", APP_ID, NETWORK_ID, SEM, TENANT_ID, CE)
}

func getTunnelId(ip, port, networkId, tenantId string) string {
	return fmt.Sprintf("%s-%s.template-rdp.%s.%s.seapp.appaegis.tunnel", ip, port, networkId, tenantId)
}

type Client struct {
	Index    int
	ServerIp string
	RunFor   time.Duration
	Jwt      string
}

type Message struct {
	Op   string
	Args []string
}

func (c *Client) Connect(wg *sync.WaitGroup) {
	defer wg.Done()

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true, ServerName: CE}

	headers := http.Header{
		"Cookie": {fmt.Sprintf("appauthz=%s", c.Jwt)},
	}
	vals := url.Values{}
	host := getTunnelId(c.ServerIp, "3389", NETWORK_ID, TENANT_ID)
	vals.Set("hostname", host)
	vals.Set("scheme", "rdp")
	vals.Set("ignore-cert", "true")
	vals.Set("username", fmt.Sprintf("user%d", c.Index))
	vals.Set("password", "Aa123456")
	vals.Set("width", "700")
	vals.Set("height", "577")
	vals.Set("color-depth", "24")
	vals.Set("enable-wallpaper", "true")
	vals.Set("enable-drive", "true")
	vals.Set("userId", "heybruce+qatest@gmail.com")
	vals.Set("appId", APP_ID)
	vals.Set("tenantId", TENANT_ID)
	vals.Set("gateway-hostname", SEM)
	vals.Set("gateway-port", "7081")

	conn, resp, err := dialer.Dial(fmt.Sprintf("wss://%s/rdpws/websocket-tunnel?%s", CE, vals.Encode()), headers)
	body, _ := httputil.DumpResponse(resp, true)
	logrus.Infof("%s", body)

	if err != nil {
		logrus.Errorf("Dial websocket failed %v", err)
		return
	}
	defer conn.Close()

	start := time.Now()
	// Launch a goroutine to read messages from the WebSocket connection and log the received messages
	go func() {
		for {
			_, data, e := conn.ReadMessage()
			if e == nil {
				if m, e := parseMessage(data); e == nil {
					logrus.Infof("Receive command op %s, args %v", m.Op, m.Args)

				} else {
					logrus.Errorf("Parse message failed: %#v, data %s. Duration: %v ms", e, string(data), time.Since(start).Milliseconds())
				}
			} else {
				logrus.Errorf("WebSocket connection failed: %v, Duration: %v ms", e, time.Since(start).Milliseconds())
				return
			}
		}
	}()
	// Simulate mouse movement
	x := 1
	y := 1
	reverse := false
	for {
		logrus.Infof("Sending mouse movement...User %d write mouse movement %d %d", c.Index, x, y)

		ins := guac.NewInstruction("mouse", []string{fmt.Sprintf("%d", x), fmt.Sprintf("%d", y)}...)

		// Start timer before sending the message
		requestStart := time.Now()

		e := conn.WriteMessage(websocket.TextMessage, ins.Byte())
		if e != nil {
			logrus.Errorf("User %v write message to guac failed. Response: %v", c.Index, e)
			break
		} else {
			// Measure the time after successfully sending the message
			requestDuration := time.Since(requestStart).Nanoseconds()
			logrus.Infof("User %v successfully wrote message to guac. Request Response Time: %f ms", c.Index, float64(requestDuration)/1e6)
		}

		time.Sleep(1000 * time.Millisecond)
		diff := 1
		if reverse {
			diff = -1
		}
		x = x + diff
		y = y + diff
		if x >= 500 {
			reverse = true
		} else if x <= 50 {
			reverse = false
		}
		if time.Since(start) > c.RunFor {
			logrus.Infof("User %d run for %v, stopped", c.Index, c.RunFor)
			break
		}
	}
}

func parseMessage(data []byte) (*Message, error) {
	ins := bytes.Split(data, []byte{','})
	if len(ins) < 1 {
		return nil, fmt.Errorf("instruction not found")
	}
	opPart := ins[0]
	i := bytes.IndexByte(opPart, '.')
	op := opPart[i+1:]
	var r Message
	r.Op = string(op)
	// logrus.Infof("op %s", r.Op)
	return &r, nil
}
