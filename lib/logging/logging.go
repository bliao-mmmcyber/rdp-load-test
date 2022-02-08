package logging

import (
	"encoding/json"
	"fmt"
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
	TenantId        string    `json:"tenantId"`
	Email           string    `json:"email"`
	AppName         string    `json:"appName"`
	ClientIp        string    `json:"clientIp"`
	S3Key           string    `json:"s3key"`
	EnableRecording bool      `json:"enableRecording"`
	StartTime       time.Time `json:"startTime"`
	Sku             string    `json:"sku"`
}

func (l *LoggingInfo) GetRecordingFileName() string {
	return fmt.Sprintf("%s-%s", l.Email, l.S3Key)
}

func NewLoggingInfo(tenantId, email, appName, clientIp, s3key, sku string, enableRecording bool) LoggingInfo {
	return LoggingInfo{
		TenantId:        tenantId,
		Email:           email,
		AppName:         appName,
		ClientIp:        clientIp,
		S3Key:           s3key,
		EnableRecording: enableRecording,
		StartTime:       time.Now(),
		Sku:             sku,
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
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}

func LogRecording(loggingInfo LoggingInfo, key string, bucket, keyId, storageType, region string) {
	recordingLogger.Info(
		"rdp-session",
		zap.Time("ts", loggingInfo.StartTime),
		zap.String("tenant", loggingInfo.TenantId),
		zap.String("username", loggingInfo.Email),
		zap.String("app_name", loggingInfo.AppName),
		zap.String("file_key", key),
		zap.String("bucket", bucket),
		zap.String("region", region),
		zap.String("client_ip", loggingInfo.ClientIp),
		zap.String("key_id", keyId),
		zap.String("storage_type", storageType),
	)
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
