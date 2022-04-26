package main

import (
	"github.com/wwt/guac/stresstest"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	userCount    = 15
	launchPeriod = 5 * time.Second
	SERVERS      = []string{"52.87.253.122"}
	index        = 0
	runFor       = 3 * time.Minute
	jwt          = "eyJraWQiOiJJRjNrbUM2OEZ4XC9NeEQ2MFRidXVLRkt4eVJDUEtHbGI3czBrN2RXNTgrND0iLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiI1YzkxNmY3OC0wYzRjLTRhNDAtYjFiMy00MDlhYWUwOWE5ZWQiLCJjb2duaXRvOmdyb3VwcyI6WyJzdXBlcnVzZXIiXSwiZW1haWxfdmVyaWZpZWQiOnRydWUsInN1cGVydXNlcnRlbmFudGlkIjoiOTM0ZWIyYzAtODczZi00MjA0LWFiYTctMGY2M2MzZjVmMzcyIiwiY29nbml0b1VzZXJFbWFpbCI6ImtjaHVuZ0BhcHBhZWdpcy5jb20iLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAudXMtZWFzdC0xLmFtYXpvbmF3cy5jb21cL3VzLWVhc3QtMV96WWx3Mlg4bEIiLCJ1c2Vycm9sZSI6InN1cGVydXNlciIsImNvZ25pdG86dXNlcm5hbWUiOiI1YzkxNmY3OC0wYzRjLTRhNDAtYjFiMy00MDlhYWUwOWE5ZWQiLCJza3VUeXBlIjoiUHJvZmVzc2lvbmFsIiwib3JpZ2luX2p0aSI6Ijk3OTY0MGUzLWM5MTUtNDU1Yi05ODY2LTY3YzUwMDFlYTk1MiIsImF1ZCI6IjU5ZmpxNG5xOGEzOWU2cTBiNGc1aHFqZWhtIiwiZXZlbnRfaWQiOiIyMTAzNzY1Ni00YTgyLTRjNDItODVhYi03ZGZkYmJkNTVjMTgiLCJ0b2tlbl91c2UiOiJpZCIsImF1dGhfdGltZSI6MTY1MjA3NjczMSwiZXhwIjoxNjUyMDgwMzMxLCJpYXQiOjE2NTIwNzY3MzEsImp0aSI6IjA2NThlZDAzLTdjMDQtNGRjMi05NmI5LWI1NDJiNGEwN2YyZCIsImVtYWlsIjoia2NodW5nQGFwcGFlZ2lzLmNvbSJ9.ZJVREjxKQPXxF1dN00dCCiivN5DNaOEvyyI5mwkaFHceAfUoJmMlL47lCnGYQtNi4hF-U_VuV_JTC717qtVdyk-l6WI78Wf6_pRODe7hpURWcwKNlaZmfkZJFSujIVQMqXFN97gkmrXaYp29MRfiRyxV3v0ChKjeYUSv9uTjeUsaetdEE-vTK3WDuVPIPErObc1vRWp4qwncV2fKAzJoW4Q9_Zo7Uv-Y1SqBogPdsRV2tg30dUWkiQPATI1hgRrmQQdPDZRa_zVSw-HOoPfQ1VpiN-gzhhDbHd9FUJLc2nJALLQiJAfr3T0I1-cn0hofdFB3DU0IEMFFKqPhB7EAhg"
)

func init() {
	if os.Getenv("USE_COUNT") != "" {
		userCount, _ = strconv.Atoi(os.Getenv("USER_COUNT"))
	}
	if os.Getenv("LAUNCH_PERIOD") != "" {
		launchPeriod, _ = time.ParseDuration(os.Getenv("LAUNCH_PERIOD"))
	}
	if os.Getenv("HOSTNAME") != "" {
		strs := strings.Split(os.Getenv("HOSTNAME"), "-")
		index, _ = strconv.Atoi(strs[1])
	}
	if os.Getenv("RUN_FOR") != "" {
		runFor, _ = time.ParseDuration(os.Getenv("RUN_FOR"))
	}
	if os.Getenv("JWT") != "" {
		jwt = os.Getenv("JWT")
	}
}

func main() {

	var wg sync.WaitGroup
	for i := 0; i < userCount; i++ {
		c := stresstest.Client{Index: i + 1, ServerIp: SERVERS[index], RunFor: runFor, Jwt: jwt}
		time.Sleep(launchPeriod)
		wg.Add(1)
		go c.Connect(&wg)
	}
	wg.Wait()
}
