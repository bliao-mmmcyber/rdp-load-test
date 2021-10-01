package logging

import (
	"encoding/json"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
var recordingLogger *zap.Logger

// Action is user action, the log object
type Action struct {
	AppType   string   `json:"app_type"`
	AppTag    string   `json:"app_tag"`
	TenantID  string   `json:"tenantID"`
	AppID     string   `json:"appID"`
	RoleIDs   []string `json:"roleIDs,omitempty"`
	UserEmail string   `json:"userEmail"`
	Username  string   `json:"username"`
	FileCount int      `json:"fileCount,omitempty"`
	ClientIP  string   `json:"client_ip"`
}

type LoggingInfo struct {
	TenantId        string
	Email           string
	AppName         string
	ClientIp        string
	S3Key           string
	EnableRecording bool
}

func NewLoggingInfo(tenantId, email, appName, clientIp, s3key string, enableRecording bool) LoggingInfo {
	return LoggingInfo{
		TenantId:        tenantId,
		Email:           email,
		AppName:         appName,
		ClientIp:        clientIp,
		S3Key:           s3key,
		EnableRecording: enableRecording,
	}
}

// Init manually create report file
func Init() {
	reportFile, err := os.OpenFile(LOG_FILE, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		logrus.Fatal(err)
	}
	logger = log.New(reportFile, "", 0)

	recordingLogger, _ = NewSessionRecordingLogger()
}

func NewSessionRecordingLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	os.OpenFile("/var/log/appaegis/guac_recordings.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)

	cfg.OutputPaths = []string{
		"/var/log/appaegis/guac_recordings.log",
	}
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}

func LogRecording(loggingInfo LoggingInfo) {
	recordingLogger.Info(
		"rdp-session",
		zap.String("tenant", loggingInfo.TenantId),
		zap.String("username", loggingInfo.Email),
		zap.String("app_name", loggingInfo.AppName),
		zap.String("s3Key", loggingInfo.S3Key),
		zap.String("client_ip", loggingInfo.ClientIp))
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
