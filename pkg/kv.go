package guac

import (
	"github.com/appaegis/golang-common/pkg/cache"
	"github.com/appaegis/golang-common/pkg/config"
	"github.com/sirupsen/logrus"
	"os"
)

var kv cache.SimpleCache
var GuacIp = "127.0.0.1"

func init() {
	kv = cache.NewRedisStore(config.GetRedisEndPoint(), cache.SimpleCacheConfiguration{
		Prefix: "/dplocal/kv/",
	})

	if os.Getenv("POD_IP") != "" {
		GuacIp = os.Getenv("POD_IP")
	}
	logrus.Infof("init redis done, guac ip %s", GuacIp)

}
