package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
	guac "github.com/wwt/guac/pkg"
)

func main() {
	logging.Init()

	podName := os.Getenv("POD_NAME")

	inK8s := podName != ""
	index := 0
	var err error
	if inK8s {
		strs := strings.Split(podName, "-")
		indexStr := strs[len(strs)-1]
		index, err = strconv.Atoi(indexStr)
		if err != nil {
			panic(err)
		}
	}
	logrus.Infof("index %d", index)
	guac.EncodeRecording(index)
}
