package guac

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
	"os"
	"strconv"
)

var db *redis.Client
var ctx context.Context

const queueName = "recording-queue"

var theNumberOfQueues int

var count int

const REDIS_PWD = "Appaegis1234"

func init() {
	ctx = context.Background()

	if os.Getenv("POD_IP") != "" {
		db = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       "mymaster",
			SentinelAddrs:    []string{"redis:26379"},
			Password:         REDIS_PWD,
			SentinelPassword: REDIS_PWD,
		})
	} else {
		db = redis.NewClient(&redis.Options{
			Addr:     "127.0.0.1:6379",
			Password: "Appaegis1234", // no password set
			DB:       0,              // use default DB
		})
	}

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
	logrus.Infof("push %s to queue %d", recording.S3Key, index)

	data, _ := json.Marshal(recording)
	_, err := db.RPush(ctx, GetQueueName(index), string(data)).Result()
	if err != nil {
		logrus.Errorf("push to redis failed %v", err)
		return
	}
}

func PeekFromQueue(index int) *logging.LoggingInfo {
	r, err := db.LRange(ctx, GetQueueName(index), 0, 0).Result()
	if err != nil {
		logrus.Errorf("lrange redis failed %v", err)
		return nil
	}
	if len(r) > 0 {
		var result logging.LoggingInfo
		if e := json.Unmarshal([]byte(r[0]), &result); e == nil {
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
	result, e := db.LPop(ctx, GetQueueName(index)).Result()
	if e != nil {
		logrus.Infof("pop result %s, e: %v", result, e)
	}
}

func GetQueueName(index int) string {
	return fmt.Sprintf("%s-%d", queueName, index)
}
