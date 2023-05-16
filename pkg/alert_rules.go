package guac

import (
	"github.com/appaegis/golang-common/pkg/dynamodbcli"
	"golang.org/x/net/websocket"
)

// AlertRuleData monitor policy alert rule data and count
type AlertRuleData struct {
	RuleID     string   `json:"id"`
	EventTypes []string `json:"events"`
	Action     string   `json:"action"`
	Condition  struct {
		FrequencyValue     int      `json:"frequencyValue"`
		FrequencyRate      string   `json:"frequencyRate"`
		Locations          []string `json:"locations"`
		LocationFilterType string   `json:"locationFilterType"`
	} `json:"condition"`
	SessionCount int
}

type SessionCommonData struct {
	TenantID         string
	AppID            string
	Email            string
	IDToken          string
	ClientIsoCountry string
	ClientIP         string
	AppName          string
	RoleIDs          []string
	SessionStartTime int64

	Recording         bool
	PolicyID          string
	PolicyName        string
	MonitorPolicyId   string
	MonitorPolicyName string
	MonitorRules      map[string]*dynamodbcli.MonitorPolicyRule

	RdpSessionId string
	GuacdAddr    string
	Websocket    *websocket.Conn
}
