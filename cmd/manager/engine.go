package main

import (
	conf "github.com/kuberik/kuberik/pkg/engine/config"
	"github.com/kuberik/kuberik/pkg/engine/runtime/scheduler"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func initEngine(config *rest.Config, client client.Client, namespace string) {
	conf.InitConfig(config)
	conf.InitClient(client)
	scheduler.InitEngine()
}
