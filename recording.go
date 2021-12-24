package guac

import (
	"fmt"
	"github.com/appaegis/golang-common/pkg/storage"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/env"
	"github.com/wwt/guac/lib/logging"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

func AddEncodeRecoding(loggingInfo logging.LoggingInfo) {
	logrus.Infof("add encoding %s", loggingInfo.S3Key)
	PushToQueue(loggingInfo)
}

func EncodeRecording(index int) {

	for {
		info := PeekFromQueue(index)
		if info != nil {
			logrus.Infof("handle %#v form queue %d", info, index)
			if _, e := os.Stat(fmt.Sprintf("/efs/rdp/%s", info.GetRecordingFileName())); e != nil {
				logrus.Infof("file %s not found, skip", info.GetRecordingFileName())
				PopFromQueue(index)
			} else {
				Encode(*info)
				PopFromQueue(index)
			}
		} else {
			time.Sleep(5 * time.Second)
		}
	}
}

func Encode(loggingInfo logging.LoggingInfo) {
	if loggingInfo.EnableRecording == false {
		return
	}

	count := 0
	for {
		count++
		time.Sleep(5 * time.Second)

		//if guac process is stopped in the middle of transcoding
		//we should delete the old temp file and do it again
		os.Remove(fmt.Sprintf("/efs/rdp/%s.mp4", loggingInfo.GetRecordingFileName()))
		os.Remove(fmt.Sprintf("/efs/rdp/%s.m4v", loggingInfo.GetRecordingFileName()))

		output, err := exec.Command("guacenc", fmt.Sprintf("/efs/rdp/%s", loggingInfo.GetRecordingFileName())).CombinedOutput()
		logrus.Infof("encode result %s, err %v", output, err)
		if err != nil {
			return
		}
		if count >= 3 {
			logrus.Infof("retry for 3 times, ignore")
			return
		}
		if err == nil && !strings.Contains(string(output), "All files encoded successfully") {
			logrus.Infof("encode %s failed, try again", loggingInfo.GetRecordingFileName())
			continue
		} else {
			break
		}
	}

	//ffmpeg -i c57fc449-c352-4efb-8501-b5203eaaafdb.m4v -vcodec libx264 -acodec aac output2.mp4
	command := fmt.Sprintf("ffmpeg -i /efs/rdp/%s.m4v -vcodec libx264 -acodec aac /efs/rdp/%s.mp4", loggingInfo.GetRecordingFileName(), loggingInfo.GetRecordingFileName())
	strs := strings.Split(command, " ")
	output, _ := exec.Command(strs[0], strs[1:]...).CombinedOutput()
	logrus.Infof("ffmpeg output %s", output)

	// RDP auth error still have m4v file
	// we check valid recording by ffmpeg result
	if !strings.Contains(string(output), "video:0kB") {

		f1, err := os.OpenFile(fmt.Sprintf("/efs/rdp/%s.mp4", loggingInfo.GetRecordingFileName()), os.O_RDONLY, 0744)
		if err != nil {
			logrus.Errorf("cannot open file %s.mp4", loggingInfo.GetRecordingFileName())
			return
		}
		tag := url.QueryEscape(fmt.Sprintf("sku=%s", loggingInfo.Sku))
		s, appaegis := storage.GetStorageByTenantId(loggingInfo.TenantId, env.Region)
		key := fmt.Sprintf("rdp/%s/%s/%s.mp4", loggingInfo.TenantId, loggingInfo.Email, loggingInfo.S3Key)
		if appaegis {
			key = fmt.Sprintf("%s/%s/%s.mp4", loggingInfo.TenantId, loggingInfo.Email, loggingInfo.S3Key)
		}
		s.UploadRdp(key, f1, tag)

		logging.LogRecording(loggingInfo, key, s.GetRdpBucket(), s.GetKeyId(), s.GetStorageType())
	}
	os.Remove(fmt.Sprintf("/efs/rdp/%s.mp4", loggingInfo.GetRecordingFileName()))
	os.Remove(fmt.Sprintf("/efs/rdp/%s.m4v", loggingInfo.GetRecordingFileName()))
	os.Remove(fmt.Sprintf("/efs/rdp/%s", loggingInfo.GetRecordingFileName()))

}
