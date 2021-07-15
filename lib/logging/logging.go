package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	LOG_FILE = "/var/log/appaegis/appaegis_guac.log"
)

var reportFile *os.File
var systemLogFile *os.File
var logger *log.Logger

// Action is user action, the log object
type Action struct {
	AppType      string   `json:"app_type"`
	AppTag       string   `json:"app_tag"`
	TenantID     string   `json:"tenantID"`
	AppID        string   `json:"appID"`
	RoleIDs      []string `json:"roleIDs,omitempty"`
	UserEmail    string   `json:"userEmail"`
	Username     string   `json:"username"`
	FileCount    int      `json:"fileCount,omitempty"`
}

// Init manually create report file
func Init() {
	reportFile, err := os.OpenFile(LOG_FILE, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		logrus.Fatal(err)
	}
	logger = log.New(reportFile, "", 0)
}

func Log(action Action) {
	action.AppType = "guac"
	data, err := json.Marshal(action)
	if err != nil {
		logrus.Errorf("unmarshall failed %s", err.Error())
		return
	}
	now := time.Now().Format("2006-01-02T15:04:05.000Z")
	logger.Printf("%s %s\n", now, string(data))
}

func Close() {
	if reportFile != nil {
		reportFile.Close()
	}
}
