package stresstest

import (
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
	APP_ID     = "0caed39b-e7d7-49c5-a463-951c536cee1b"
	NETWORK_ID = "3eb42b35-e4a0-493a-8e66-9485883610bb"
	SEM        = "10.0.1.12"
	TENANT_ID  = "934eb2c0-873f-4204-aba7-0f63c3f5f372"
	CE         = "dev.ce.appaegistest.com"
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
	vals.Set("userId", "kchung@appaegis.com")
	vals.Set("appId", APP_ID)
	vals.Set("tenantId", TENANT_ID)
	vals.Set("gateway-hostname", SEM)
	vals.Set("gateway-port", "7081")

	conn, resp, err := dialer.Dial(fmt.Sprintf("wss://%s/rdpws/websocket-tunnel?%s", CE, vals.Encode()), headers)
	body, _ := httputil.DumpResponse(resp, true)
	logrus.Infof("%s", body)

	if err != nil {
		logrus.Errorf("dial websocket failed %v", err)
		return
	}

	defer conn.Close()

	go func() {
		for {
			_, _, e := conn.ReadMessage()
			if e == nil {
				logrus.Infof("client received data")
				// logrus.Infof("receive %s", msg)
			} else {
				logrus.Errorf("ws failed %v", e)
				return
			}
		}
	}()

	start := time.Now()
	x := 50
	y := 50
	reverse := false
	for {
		logrus.Infof("user %d write mouse %d %d", c.Index, x, y)
		ins := guac.NewInstruction("mouse", []string{fmt.Sprintf("%d", x), fmt.Sprintf("%d", y)}...)
		e := conn.WriteMessage(websocket.TextMessage, ins.Byte())
		if e != nil {
			logrus.Errorf("write message to guac failed %v", e)
			break
		}
		time.Sleep(100 * time.Millisecond)
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
			break
		}
	}
}
