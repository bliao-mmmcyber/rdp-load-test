package guac

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/appaegis/golang-common/pkg/config"
	"github.com/appaegis/golang-common/pkg/queue"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
)

const queueName = "recording-queue"

var theNumberOfQueues int

var count int

var q queue.QueueService

func init() {
	q = queue.NewRedisQueueService(config.GetRedisEndPoint())
	if os.Getenv("NUMBER_OF_TRANSCODING_QUEUE") != "" {
		count, e := strconv.Atoi(os.Getenv("NUMBER_OF_TRANSCODING_QUEUE"))
		if e != nil {
			panic(e)
		}
		theNumberOfQueues = count
	} else {
		theNumberOfQueues = 1
	}
	logrus.Infof("init redis done, the number of queue %d", theNumberOfQueues)
}

func PushToQueue(recording logging.LoggingInfo) {
	if !recording.EnableRecording {
		return
	}

	count++
	index := count % theNumberOfQueues
	logrus.Infof("push %s to queue %d, queue name %s", recording.S3Key, index, GetQueueName(index))

	data, _ := json.Marshal(recording)
	err := q.PushToQueue(GetQueueName(index), string(data))
	if err != nil {
		logrus.Errorf("push to redis failed %v", err)
		return
	}
}

func PeekFromQueue(index int) *logging.LoggingInfo {
	msg, err := q.PeekFromQueue(GetQueueName(index))
	if err != nil {
		logrus.Errorf("peek from queue failed %v", err)
		return nil
	}
	if msg != "" {
		var result logging.LoggingInfo
		if e := json.Unmarshal([]byte(msg), &result); e == nil {
			return &result
		} else {
			logrus.Errorf("unmarshall loggingInfo failed %v", e)
			return nil
		}
	}
	return nil
}

func PopFromQueue(index int) {
	logrus.Infof("pop from queue %d", index)
	_, e := q.PopFromQueue(GetQueueName(index))
	if e != nil {
		logrus.Infof("pop e: %v", e)
	}
}

func GetQueueName(index int) string {
	return fmt.Sprintf("%s-%s-%d", queueName, config.GetDeployVer(), index)
}
