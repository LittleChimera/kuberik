package scheduler

import (
	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
)

// Engine is a Engine used to schedule Actions throughout program execution
var engine Scheduler

// Scheduler implements a way for launching Actions
type Scheduler interface {
	Run(play *corev1alpha1.Play, frameID string) error
}

// Run executes an Action of a single Frame of a Play
func Run(play *corev1alpha1.Play, frameID string) error {
	if engine == nil {
		panic("Engine not initialized. Use runtime.InitEngine() to initialize it.")
	}
	return engine.Run(play, frameID)
}

// InitEngine initializes Engine Engine used to schedule Actions througout program execution
func InitEngine(e Scheduler) {
	engine = e
}
