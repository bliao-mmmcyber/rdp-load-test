package guac

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
