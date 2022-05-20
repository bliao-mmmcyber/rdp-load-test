package guac

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"time"
)

var clientset *kubernetes.Clientset
var NAMESPACE = "appaegis"

func InitK8S() {
	var e error
	var cfg *rest.Config
	if os.Getenv("POD_IP") != "" {
		cfg, e = rest.InClusterConfig()
	} else {
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		logrus.Infof("kubect config %s", kubeconfig)
		cfg, e = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if e != nil {
		panic(e)
	}
	clientset, e = kubernetes.NewForConfig(cfg)
	if e != nil {
		panic(e)
	}
	if os.Getenv("POD_NAMESPACE") != "" {
		NAMESPACE = os.Getenv("POD_NAMESPACE")
	}
}

// GetGuacdTarget use guacd-service cannot force sharing session invite connecting to the same guacd
func GetGuacdTarget() (string, error) {
	endpoints, e := clientset.CoreV1().Endpoints(NAMESPACE).Get(context.Background(), "guacd-service", metav1.GetOptions{})
	if e != nil {
		logrus.Error("get endpoints failed %v", e)
		return "", e
	}
	if len(endpoints.Subsets) <= 0 {
		logrus.Error("endpoints size = 0")
		return "", fmt.Errorf("no endpoints for guacd")
	}
	rand.Seed(time.Now().Unix())
	size := len(endpoints.Subsets[0].Addresses)
	addr := endpoints.Subsets[0].Addresses[rand.Intn(size)].IP
	return addr, nil
}
