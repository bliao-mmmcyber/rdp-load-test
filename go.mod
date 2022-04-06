module github.com/wwt/guac

go 1.15

replace github.com/appaegis/golang-common => ../golang-common

require (
	github.com/appaegis/golang-common v0.0.0-20210118093202-088b8b8751c7
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/oschwald/geoip2-golang v1.5.0
	github.com/prometheus/client_golang v1.11.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	go.uber.org/zap v1.17.0
	mvdan.cc/gofumpt v0.3.1
)
