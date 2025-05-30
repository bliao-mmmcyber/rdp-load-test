package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/pkg/session"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	LOG_FILE = "/var/log/appaegis/appaegis_guac.log"
)

var (
	reportFile      *os.File
	logger          *log.Logger
	recordingLogger *zap.Logger
)

// Action is user action, the log object
type Action struct {
	Session *session.SessionCommonData `json:"-"`

	AppTag            string `json:"app_tag"`
	AppType           string `json:"app_type"`
	ProductType       string `json:"product_type"`
	TenantID          string `json:"tenantID"`
	AppID             string `json:"appID"`
	AppName           string `json:"appName"`
	RdpSessionId      string `json:"rdpSessionId"`
	Recording         bool   `json:"recording"`
	MonitorPolicyId   string `json:"monitorpolicyid"`
	MonitorPolicyName string `json:"monitorpolicyname"`

	PolicyID        string   `json:"policyid"`
	PolicyName      string   `json:"policyname"`
	UserEmail       string   `json:"userEmail"`
	Username        string   `json:"username"`
	RemotePath      string   `json:"remotePath"`
	FileCount       int      `json:"fileCount,omitempty"`
	Files           []string `json:"files"`
	ClientIP        string   `json:"client_ip"`
	ClientPrivateIp string   `json:"client_private_ip"`
	Destination     string   `json:"dest"`

	BlockPolicyType string `json:"blockPolicyType"`
	BlockReason     string `json:"blockReason"`
}

func (a *Action) FillAttribute() {
	a.TenantID = a.Session.TenantID
	a.AppID = a.Session.AppID
	a.AppName = a.Session.AppName
	a.RdpSessionId = a.Session.RdpSessionId
	a.MonitorPolicyId = a.Session.MonitorPolicyId
	a.MonitorPolicyName = a.Session.MonitorPolicyName
}

type LoggingInfo struct {
	TenantId        string    `json:"tenantId"`
	Email           string    `json:"email"`
	AppName         string    `json:"appName"`
	ClientIp        string    `json:"clientIp"`
	ClientPrivateIp string    `json:"clientPrivateIp"`
	S3Key           string    `json:"s3key"`
	EnableRecording bool      `json:"enableRecording"`
	StartTime       time.Time `json:"startTime"`
	Sku             string    `json:"sku"`
	SessionId       string    `json:"sessionid"`
}

func (l *LoggingInfo) GetRecordingFileName() string {
	return fmt.Sprintf("%s-%s", l.Email, l.S3Key)
}

func NewLoggingInfo(tenantId, email, appName, clientIp, s3key, sku string, enableRecording bool, clientPrivateIp string) LoggingInfo {
	return LoggingInfo{
		TenantId:        tenantId,
		Email:           email,
		AppName:         appName,
		ClientIp:        clientIp,
		ClientPrivateIp: clientPrivateIp,
		S3Key:           s3key,
		EnableRecording: enableRecording,
		StartTime:       time.Now(),
		Sku:             sku,
	}
}

func init() {
	logger = log.Default() // default logger for unit test
}

// Init manually create report file
func Init() {
	reportFile, err := os.OpenFile(LOG_FILE, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o755)
	if err != nil {
		logrus.Fatal(err)
	}
	logger = log.New(reportFile, "", 0)

	recordingLogger, _ = NewSessionRecordingLogger()
}

func NewSessionRecordingLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	_, _ = os.OpenFile("/var/log/appaegis/guac_recordings.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o744)

	cfg.OutputPaths = []string{
		"/var/log/appaegis/guac_recordings.log",
	}
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}

func LogRecording(loggingInfo LoggingInfo, key string, bucket, keyId, storageType, region, sessionid string) {
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
		zap.String("client_private_ip", loggingInfo.ClientPrivateIp),
		zap.String("key_id", keyId),
		zap.String("storage_type", storageType),
		zap.String("sessionid", sessionid),
	)
}

func Log(action Action) {
	action.AppType = "rdp"
	action.ProductType = "Portal"
	action.FillAttribute()
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
