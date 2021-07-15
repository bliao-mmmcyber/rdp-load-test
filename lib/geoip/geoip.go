package geoip

import (
	"net"
	"strings"

	geoip2 "github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
)

var db *geoip2.Reader
// Init initialize geolite db
func Init() {
	dbPath := "/home/appaegis/guac-assets/GeoLite2-Country.mmdb"
	mmdb, err := geoip2.Open(dbPath)
	if err != nil {
		log.Errorf("failed to open geoip db: %s", err.Error())
	} else {
		log.Infof("success to open geoip db: %s", dbPath)
		db = mmdb
	}
}

func GetIpIsoCode(ipStr string) (ISOCode string) {
	if db == nil {
		log.Infof("db is nil")
		return ""
	}
	ipParts := strings.Split(ipStr, ":")
	ipAddr := ipParts[0]
	ip := net.ParseIP(ipAddr)
	record, err := db.Country(ip)
	if err != nil {
		log.Errorf("%s", err.Error())
		return ""
	}
	return record.Country.IsoCode
}
