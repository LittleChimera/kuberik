package shell

import (
	"io"
	"os/exec"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
)

type Shell struct{}

func (s *Shell) Run(namespace, name string, e corev1alpha1.Exec) (io.Reader, chan int, error) {
	reader, writer := io.Pipe()
	result := make(chan int)

	var args []string
	var command string
	args = append(args, e.Template.Spec.Containers[0].Args...)
	if execCommand := e.Template.Spec.Containers[0].Command; len(execCommand) > 0 {
		args = append(execCommand[1:], args...)
		command = execCommand[0]
	}
	cmd := exec.Command(command, args...)
	cmd.Stdout = writer
	cmd.Stderr = writer

	cmd.Start()

	go func() {
		err := cmd.Wait()
		writer.Close()
		// TODO This looks so stupid
		if _, ok := err.(*exec.ExitError); ok {
			result <- 1
		} else {
			result <- 0
		}
		close(result)
	}()

	return reader, result, nil
}

func (r *Shell) UpdatePlayPhase(play corev1alpha1.Play, phase corev1alpha1.PlayPhaseType) error {
	return nil
}
