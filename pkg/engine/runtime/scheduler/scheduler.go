package scheduler

import (
	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/config"
	"github.com/kuberik/kuberik/pkg/engine/runtime/scheduler/kubernetes"
)

// Engine is a Engine used to schedule Actions throughout program execution
var Engine Scheduler

// Scheduler implements a way for launching Actions
type Scheduler interface {
	Run(play *corev1alpha1.Play, frameID string) error
}

// InitEngine initializes Engine Engine used to schedule Actions througout program execution
func InitEngine() {
	Engine = kubernetes.NewScheduler(config.Config)
}
