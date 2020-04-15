package config

import (
	"os"

	"github.com/kuberik/kuberik/pkg/randutils"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO remove
var Config *rest.Config

// TODO remove
var Client client.Client

var RunnerID string
var Host string

func InitConfig(c *rest.Config) {
	Config = c
	RunnerID = randutils.Rand()
}

func InitClient(c client.Client) {
	Client = c
}

func init() {
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		// Running in the cluster - listen on all interfaces
		Host = "0.0.0.0"
	} else {
		// Running on development machine - use localhost to avoid MacOS firewall prompts
		Host = "127.0.0.1"
	}
}
