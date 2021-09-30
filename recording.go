package guac

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
	"os"
	"os/exec"
	"strings"
	"time"
)

var S3Client *s3.S3
var S3Uploader *s3manager.Uploader
var BUCKET_NAME string

var recordingCh = make(chan logging.LoggingInfo, 1024)

func init() {
	REGION := os.Getenv("CE_COG_REGION")
	DEPLOY_ENV := os.Getenv("DEPLOY_ENV")
	logrus.Infof("init with env %s, region %s", DEPLOY_ENV, REGION)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(REGION)},
	)
	if err != nil {
		logrus.Errorf("error create aws session %v", err)
		return
	}

	S3Client = s3.New(sess)
	S3Uploader = s3manager.NewUploaderWithClient(S3Client)

	BUCKET_NAME = fmt.Sprintf("appaegis-rdp-%s", DEPLOY_ENV)
	_, err = S3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(BUCKET_NAME),
	})
	if err != nil {
		logrus.Errorf("create s3 bucket failed %v", err)
	}
}

func AddEncodeRecoding(loggingInfo logging.LoggingInfo) {
	recordingCh <- loggingInfo
}

func EncodeRecording() {

	for recording := range recordingCh {
		logrus.Infof("recording %s", recording)
		go Encode(recording)
	}
}

func Encode(loggingInfo logging.LoggingInfo) {
	count := 0
	for {
		count++
		time.Sleep(5 * time.Second)

		output, err := exec.Command("guacenc", fmt.Sprintf("/efs/rdp/%s", loggingInfo.S3Key)).CombinedOutput()
		logrus.Infof("encode result %s, err %v", output, err)
		if err != nil {
			return
		}
		if count >= 3 {
			logrus.Infof("retry for 3 times, ignore")
			return
		}
		if err == nil && !strings.Contains(string(output), "All files encoded successfully") {
			logrus.Infof("encode %s failed, try again", loggingInfo.S3Key)
			continue
		} else {
			break
		}
	}

	f1, err := os.OpenFile(fmt.Sprintf("/efs/rdp/%s.m4v", loggingInfo.S3Key), os.O_RDONLY, 0744)
	if err != nil {
		logrus.Errorf("cannot open file %s.m4v", loggingInfo.S3Key)
		return
	}
	result, e := S3Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(BUCKET_NAME),
		Key:    aws.String(fmt.Sprintf("%s/%s.m4v", loggingInfo.TenantId, loggingInfo.S3Key)),
		Body:   f1,
	})
	if e != nil {
		logrus.Errorf("uplodate script file error %v", e)
	} else {
		logrus.Infof("upload result %v", result)
	}

	logging.LogRecording(loggingInfo)
	os.Remove(fmt.Sprintf("/efs/rdp/%s.m4v", loggingInfo.S3Key))
	os.Remove(fmt.Sprintf("/efs/rdp/%s", loggingInfo.S3Key))

}
