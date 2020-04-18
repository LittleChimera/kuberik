package shell

import (
	"os/exec"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
)

type Shell struct{}

func (s *Shell) Run(play *corev1alpha1.Play, frameID string) (chan int, error) {
	result := make(chan int)
	e := play.Frame(frameID).Action

	var args []string
	var command string
	args = append(args, e.Template.Spec.Containers[0].Args...)
	if execCommand := e.Template.Spec.Containers[0].Command; len(execCommand) > 0 {
		args = append(execCommand[1:], args...)
		command = execCommand[0]
	}
	cmd := exec.Command(command, args...)

	cmd.Start()

	err := cmd.Wait()
	if _, ok := err.(*exec.ExitError); ok {
		result <- 1
	} else {
		result <- 0
	}
	close(result)

	return result, nil
}
