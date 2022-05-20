package guac

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGetTarget(t *testing.T) {
	t.Skip()
	InitK8S()
	target, e := GetGuacdTarget()
	if e != nil {
		t.Errorf(e.Error())
	}
	logrus.Infof("result %s", target)
}
