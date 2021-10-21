module github.com/wwt/guac

go 1.14

replace (
	cloud.google.com/go => cloud.google.com/go v0.52.0
	github.com/appaegis/golang-common => ../golang-common
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5
	google.golang.org/api => google.golang.org/api v0.14.0
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20191216164720-4f79533eabd1
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)

require (
	github.com/appaegis/golang-common v0.0.0-20210118093202-088b8b8751c7
	github.com/aws/aws-sdk-go v1.37.33
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/go-redis/redis/v8 v8.11.4
	github.com/gorilla/websocket v1.4.2
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/oschwald/geoip2-golang v1.5.0
	github.com/prometheus/client_golang v1.10.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	go.uber.org/zap v1.16.0
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
)
