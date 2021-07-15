package env

import (
	"github.com/appaegis/golang-common/pkg/etcd"
)

var (
	// PolicyManagementHost Policy Management Host
	PolicyManagementHost string
	// PortalAPIHost Portal API Host
	PortalAPIHost string
)

// Init manaully fetch runtime environment variables
func Init() {
	etcd.NewWithEnv()
	pmRes, _ := etcd.Get("/dplocal/dp_setting/POLICY_MANAGEMENT_ENDPOINT")
	PolicyManagementHost = string(pmRes.Kvs[0].Value)

	portalAPIHostRes, _ := etcd.Get("/dplocal/dp_setting/PORTAL_API_HOST")
	PortalAPIHost = string(portalAPIHostRes.Kvs[0].Value)
}
