package scheduler

import (
	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
)

// DummyScheduler implements Scheduler interface but doesn't run any workload
type DummyScheduler struct{}

var _ Scheduler = &DummyScheduler{}

// Run implements Scheduler interface
func (s *DummyScheduler) Run(play *corev1alpha1.Play, frameID string) error {
	return nil
}
