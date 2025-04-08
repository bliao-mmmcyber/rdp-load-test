package main

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/stresstest"
)

var (
	userCount    = 30
	launchPeriod = 5 * time.Second
	SERVERS      = []string{"192.168.50.21"}
	index        = 0
	runFor       = 3 * time.Minute
	jwt          = "eyJraWQiOiJDSEhQc1g2NUF6VGYyVTZGTm03UGF5ejVKQnh0aG5tWHk2SzRZTzdrRVhJPSIsImFsZyI6IlJTMjU2In0.eyJzdWIiOiJjM2ZkN2U4My1lZjQwLTRmNGEtYjhmYy1jZWIzOGNlM2QwZjUiLCJjb2duaXRvOmdyb3VwcyI6WyJzdXBlcnVzZXIiXSwiZW1haWxfdmVyaWZpZWQiOnRydWUsInN1cGVydXNlcnRlbmFudGlkIjoiOGMxOTdlYzgtNzdkYi00NTU4LWE1N2ItYTcwODA0MGMyNTY2IiwiY29nbml0b1VzZXJFbWFpbCI6ImhleWJydWNlQGdtYWlsLmNvbSIsImlzcyI6Imh0dHBzOlwvXC9jb2duaXRvLWlkcC51cy13ZXN0LTIuYW1hem9uYXdzLmNvbVwvdXMtd2VzdC0yX29BNU1BUkpoUCIsInBob25lX251bWJlcl92ZXJpZmllZCI6ZmFsc2UsInVzZXJyb2xlIjoic3VwZXJ1c2VyIiwiY29nbml0bzp1c2VybmFtZSI6ImMzZmQ3ZTgzLWVmNDAtNGY0YS1iOGZjLWNlYjM4Y2UzZDBmNSIsImlzc3VlZFRocm91Z2hTaWxlbnRBdXRoIjoiZmFsc2UiLCJza3VUeXBlIjoiSGlnaCIsImF1ZCI6IjZtbHByM3Zocm42NGEwbGpvMDMwZnQxaWFnIiwiZXZlbnRfaWQiOiIyYzI4ZDM0Ny0zYTZkLTQ2ZTctYThkZC1hNjYzYzgxNTQ3NjIiLCJ0b2tlbl91c2UiOiJpZCIsInBlcm1pc3Npb25zIjoiRlFPbHlBPT0iLCJhdXRoX3RpbWUiOjE3NDQwODA0ODgsImV4cCI6MTc0NDA4NDA4OCwiaWF0IjoxNzQ0MDgwNDg4LCJlbWFpbCI6ImhleWJydWNlQGdtYWlsLmNvbSJ9.AnL8O0a-AGMmpHwfNztMmXb7FBvQ2TznC_Dv-teV4AuUvI00y073uaH0tgG29VxcpSL_MiXO33jw8otbgq-56ZcHA3-UXVJMmqfYL6K7_qP-hsig3Wm7qLGOe9bc5ao-PQAY2g4KA1FjTag0K9__p0xuK3V_MHkv7ARSZ9ZaYs_6GGZ7nYtLLerlSRHdAc5IFJPe5R6Me8yvdyRIJQbh4Isrmwxri9WyU6A6vPYmFc7ZM2wJRYCoxSHm7WqCk3CoaZXwi5i4jjdjowkHRYm7xBgE3YyrcJCMtxADoDgiSGpdoaG_VPv4vGPAdn4XACVAR48277AEfml-LMXf_2phxA"
)

func init() {
	if os.Getenv("USER_COUNT") != "" {
		userCount, _ = strconv.Atoi(os.Getenv("USER_COUNT"))
	}
	if os.Getenv("LAUNCH_PERIOD") != "" {
		launchPeriod, _ = time.ParseDuration(os.Getenv("LAUNCH_PERIOD"))
	}
	if os.Getenv("HOSTNAME") != "" {
		logrus.Infof("hostname %s", os.Getenv("HOSTNAME"))
		strs := strings.Split(os.Getenv("HOSTNAME"), "-")
		index, _ = strconv.Atoi(strs[1])
	}
	if os.Getenv("RUN_FOR") != "" {
		runFor, _ = time.ParseDuration(os.Getenv("RUN_FOR"))
	}
	if os.Getenv("JWT") != "" {
		jwt = os.Getenv("JWT")
	}
	logrus.Infof("count %d, period %v, index %d, run for %v", userCount, launchPeriod, index, runFor)
}

func main() {
	logrus.Infof("start running")

	var wg sync.WaitGroup
	for i := 0; i < userCount; i++ {
		c := stresstest.Client{Index: i + 1, ServerIp: SERVERS[index], RunFor: runFor, Jwt: jwt}
		time.Sleep(launchPeriod)
		wg.Add(1)
		logrus.Infof("connect client %d", c.Index)
		go c.Connect(&wg)
	}
	wg.Wait()
}
