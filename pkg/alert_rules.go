package guac

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/appaegis/golang-common/pkg/config"

	log "github.com/sirupsen/logrus"
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
	// RuleIDs ruleAction -> rule IDs
	RuleIDs map[string][]string
	// RuleID -> AlertRuleData
	Rules            map[string]*AlertRuleData
	RoleIDs          []string
	SessionStartTime int64
	ConnectionId     string
}

// response from our aggregation API
type alertRuleResult struct {
	// ruleId -> count
	Result map[string]int `json:"result"`
}

type alertRuleDataRequest struct {
	AppID          string              `json:"appId"`
	StartTimestamp int64               `json:"startTimestamp"`
	Rules          []alertRuleDataRule `json:"rules"`
}

type alertRuleDataRule struct {
	ID            string `json:"id"`
	FrequencyRate string `json:"frequencyRate"`
}

// CheckAlertRule add up count this action will incur, and check if user can perform the action
func CheckAlertRule(ses *SessionCommonData, action string, actionCount int) (canDo bool, hardBlock bool) {
	canDo = true
	hardBlock = false
	// XXX maybe we should merge into current rule check
	log.Infof("CheckAlertRuleWithCount start: %s %d", action, actionCount)
	// TODO it's possible that the frequency rate
	// is less than the time user use the app, and in that case,
	// we can/should skip make the check call to API,
	// and use local session count to do the judgement call
	// but this is for optimization anyway :)

	authToken := ses.IDToken

	var ruleData *alertRuleResult
	sessionRuleCount := make(map[string]int)
	sessionRuleLimit := make(map[string]int)
	ruleActionShouldBlock := make(map[string]bool)

	ruleIds, found := ses.RuleIDs[action]
	if !found {
		// no rules for this action, skip checking
		log.Infof("rule not found: %s", action)
		return
	}
	rules := make([]*AlertRuleData, 0)
	for _, ruleID := range ruleIds {
		rule, found := ses.Rules[ruleID]
		log.Infof("ruleId: %s, rule: %v", ruleID, rule)
		if !found {
			continue
		}
		if len(rule.Condition.Locations) > 0 {
			userCountry := ses.ClientIsoCountry
			contained := false
			for _, country := range rule.Condition.Locations {
				if country == userCountry {
					contained = true
					break
				}
			}
			shouldInclude := rule.Condition.LocationFilterType == "include"
			if shouldInclude != contained {
				continue
			}
		}
		sessionRuleCount[rule.RuleID] = rule.SessionCount
		sessionRuleLimit[rule.RuleID] = rule.Condition.FrequencyValue
		ruleActionShouldBlock[rule.RuleID] = rule.Action == "deny"

		rules = append(rules, rule)
	}
	log.Infof("rules: %v", rules)

	if len(rules) == 0 {
		// no rules for this action, skip checking
		return
	}

	ruleData = fetchAlertRuleData(authToken, ses.AppID, ses.Email, rules, ses.SessionStartTime)
	log.Infof("ruleData: %v", ruleData)
	log.Infof("sessionRuleCount: %v", sessionRuleCount)
	log.Infof("sessionRuleLimit: %v", sessionRuleLimit)
	log.Infof("ruleActionShouldBlock: %v", ruleActionShouldBlock)

	if ruleData == nil {
		// it's either there is no associated rules for this action
		// or there is some error, we let user do what they want anyway
		return
	}

	for ruleID, count := range ruleData.Result {
		sessionCount, found := sessionRuleCount[ruleID]
		if !found {
			continue
		}
		sessionLimit, found := sessionRuleLimit[ruleID]
		if !found {
			continue
		}
		shouldBlock, found := ruleActionShouldBlock[ruleID]
		if !found {
			continue
		}
		if count+sessionCount+actionCount > sessionLimit {
			canDo = false
			if shouldBlock {
				hardBlock = true
				return
			}
		}
	}

	return
}

func fetchAlertRuleData(authToken string, appID string, userName string, alertRuleData []*AlertRuleData, sessionStartTime int64) *alertRuleResult {
	log.Info("fetchAlertRuleData start")
	url := config.GetPortalApiHost() + "/rest/v1/self/app_alert/fetchAlertRuleData"

	body := bytes.Buffer{}
	payload := alertRuleDataRequest{
		AppID:          appID,
		StartTimestamp: sessionStartTime,
	}
	payload.Rules = make([]alertRuleDataRule, 0)
	for _, rule := range alertRuleData {
		payload.Rules = append(payload.Rules, alertRuleDataRule{
			ID:            rule.RuleID,
			FrequencyRate: rule.Condition.FrequencyRate,
		})
	}
	payloadString, _ := json.Marshal(payload)
	body.Write(payloadString)
	log.Infof("fetchAlertRuleData body %s", body.String())

	req, _ := http.NewRequest("POST", url, &body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("idToken", authToken)

	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		log.Errorf("failed to request alert rule aggr api %s", err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := ioutil.ReadAll(resp.Body)
		log.Errorf("failed to request alert rule aggr api %s", string(data))
		return nil
	}
	data := alertRuleResult{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Errorf("failed to decode alert rule aggr api response %s", err.Error())
		return nil
	}

	return &data
}

func IncrAlertRuleSessionCountByNumber(ses *SessionCommonData, action string, count int) {
	ruleIds, found := ses.RuleIDs[action]
	if !found {
		return
	}

	for _, ruleID := range ruleIds {
		if rule, found := ses.Rules[ruleID]; found {
			rule.SessionCount += count
			log.Infof("%s aggr count %s %d", action, ruleID, rule.SessionCount)
		}
	}
}
