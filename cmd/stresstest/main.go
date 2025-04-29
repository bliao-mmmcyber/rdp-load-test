package main

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/stresstest"
)

var (
	userCount    = 1
	launchPeriod = 5 * time.Second
	SERVERS      = []string{"192.168.50.21"}
	index        = 0
	runFor       = 3 * time.Minute
	jwt          = "eyJraWQiOiJEaUUrbTc4XC9nRVNJb2ZhVHNxWHVFeFE4aWdQam4wdU1hdTQ1ZWlwTDlOaz0iLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJkNGRjMzRkYy01ZTE4LTQ5MmQtYmNiYS1lYmRjMzczMzgyNTYiLCJjb2duaXRvOmdyb3VwcyI6WyJzdXBlcnVzZXIiXSwiZW1haWxfdmVyaWZpZWQiOnRydWUsInN1cGVydXNlcnRlbmFudGlkIjoiNTgzZWQ2ODgtYWU4ZC00ZTg1LWE2ZWQtNjY5YzMyODEwOGE0IiwiY29nbml0b1VzZXJFbWFpbCI6ImhleWJydWNlK3FhdGVzdEBnbWFpbC5jb20iLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAudXMtZWFzdC0xLmFtYXpvbmF3cy5jb21cL3VzLWVhc3QtMV9mcjEzYWtETTciLCJwaG9uZV9udW1iZXJfdmVyaWZpZWQiOmZhbHNlLCJ1c2Vycm9sZSI6InN1cGVydXNlciIsImNvZ25pdG86dXNlcm5hbWUiOiJkNGRjMzRkYy01ZTE4LTQ5MmQtYmNiYS1lYmRjMzczMzgyNTYiLCJpc3N1ZWRUaHJvdWdoU2lsZW50QXV0aCI6ImZhbHNlIiwic2t1VHlwZSI6IkhpZ2giLCJhdWQiOiI3aWtlYmkyaGd2Z2duazRkYmhyMjVsa3M4bCIsImV2ZW50X2lkIjoiYzY1YmQ4ODItNWNiYi00Mzk3LWE2NzEtOGYxM2Y4MzQ5NmQ5IiwidG9rZW5fdXNlIjoiaWQiLCJwZXJtaXNzaW9ucyI6IkZRT2x5QT09IiwiYXV0aF90aW1lIjoxNzQ1ODk2Mjc4LCJleHAiOjE3NDU5MDc2MTgsImlhdCI6MTc0NTkwNDAxOCwiZW1haWwiOiJoZXlicnVjZStxYXRlc3RAZ21haWwuY29tIn0.Z8IuIuvTA2-Bdp8MXK7tuvwErFvBtY9DgqtIF-s811_nGSgrjxlQzSHFXhC1AitaVsxYTsF9rNrbvtKBDy7CacN8yMOfpsBSij6UAIRSdZeRRr9EAfScY_JyDE84x1NQLuw5zdafS5bC-dfnYWP3M8JqSJhl294A_PzdIEeR6CVmWTSVOOPBC0SnXjN_Nrty8tK9BlsuSxZoYJBBjkNsJkhp6dWM49k-M5a9lJtoqCHSzqbbUz-GNtehvFUR2Zriq79xRzX07GCkRdZm8SXtIO1iBBZ_sxsHNlag_PhHK_1UsJfTjds-eGcxCm6Xo6KBgYaEOSuay1vF9Lj5JIo8Pw"
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
		wg.Add(1)
		c := stresstest.Client{Index: i + 1, ServerIp: SERVERS[index], RunFor: runFor, Jwt: jwt}

		// time.Sleep(launchPeriod)
		// logrus.Infof("connect client %d", c.Index)
		// go c.Connect(&wg)
		go func(client stresstest.Client) {
			defer time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			logrus.Infof("connect client %d", client.Index)
			go client.Connect(&wg)
		}(c)
	}
	wg.Wait()
}
