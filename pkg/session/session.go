package session

import (
	"time"

	"github.com/appaegis/golang-common/pkg/db_data/schema"
	"github.com/gorilla/websocket"
)

type SessionCommonData struct {
	Auth             bool
	ServerName       string
	TenantID         string
	AppID            string
	Email            string
	UserName         string
	IDToken          string
	ClientIsoCountry string
	ClientIP         string
	ClientPrivateIp  string
	AppName          string
	RoleIDs          []string
	SessionStartTime time.Time

	Recording         bool
	MonitorPolicyId   string
	MonitorPolicyName string
	MonitorRules      map[string]*schema.MonitorPolicyRule

	RdpSessionId string
	GuacdAddr    string
	Websocket    *websocket.Conn
}
