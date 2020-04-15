package scheduler

import (
	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/config"
	"github.com/kuberik/kuberik/pkg/engine/runtime/scheduler/kubernetes"
)

var Engine Scheduler

type Scheduler interface {
	Run(play *corev1alpha1.Play, frameID string) (chan int, error)
	UpdatePlayPhase(play corev1alpha1.Play, status corev1alpha1.PlayPhaseType) error
	UpdateFrameResult(play corev1alpha1.Play, ID string, result int) error
}

func InitEngine() {
	Engine = kubernetes.NewKubernetesRuntime(config.Config)
}
