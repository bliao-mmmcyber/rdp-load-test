package guac

import (
	"testing"

	"github.com/appaegis/golang-common/pkg/config"
	"github.com/sirupsen/logrus"
)

func TestSendEmail(t *testing.T) {
	config.AddConfig(config.CE_COG_REGION, "us-east-1")

	e := mailService.SendInvitation("kchung@appaegis.com", "kchung@appaegis.com", "https://dev.appaegistest.com/share_session")
	if e != nil {
		logrus.Errorf("send email failed %v", e)
		t.Fatal(e)
	}
}
