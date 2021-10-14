package env

import (
	"github.com/appaegis/golang-common/pkg/etcd"
)

var (
	// PolicyManagementHost Policy Management Host
	PolicyManagementHost string
	// PortalAPIHost Portal API Host
	PortalAPIHost     string
	DLPClientEndPoint string
	Region            string
)

// Init manaully fetch runtime environment variables
func Init() {
	etcd.NewWithEnv()
	pmRes, _ := etcd.Get("/dplocal/dp_setting/POLICY_MANAGEMENT_ENDPOINT")
	PolicyManagementHost = string(pmRes.Kvs[0].Value)

	portalAPIHostRes, _ := etcd.Get("/dplocal/dp_setting/PORTAL_API_HOST")
	PortalAPIHost = string(portalAPIHostRes.Kvs[0].Value)

	DLPClientEndPointRes, _ := etcd.Get("/dplocal/dp_setting/DLP_CLIENT_HOST")
	DLPClientEndPoint = string(DLPClientEndPointRes.Kvs[0].Value)

	RegionRes, _ := etcd.Get("/dplocal/dp_setting/CE_COG_REGION")
	Region = string(RegionRes.Kvs[0].Value)
}
