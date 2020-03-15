package cmd

import (
	"fmt"

	"github.com/kuberik/kuberik/pkg/generated/clientset/versioned/typed/core/v1alpha1"
	"k8s.io/client-go/kubernetes"

	// Include all auth clients: gcp, oidc...
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	cfg        *rest.Config
	client     *v1alpha1.CoreV1alpha1Client
	clientCfg  *api.Config
	kubeClient *kubernetes.Clientset
	namespace  string
)

func init() {
	var err error
	cfg, err = config.GetConfig()
	if err != nil {
		fmt.Println(err)
	}
	client, err = v1alpha1.NewForConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}
	clientCfg, err = clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		fmt.Println(err)
	}
	kubeClient, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}
	namespace = clientCfg.Contexts[clientCfg.CurrentContext].Namespace
	if namespace == "" {
		namespace = "default"
	}
}
