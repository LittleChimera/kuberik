package scheduler

import (
	"bytes"
	"fmt"
	"io"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/config"
	"github.com/kuberik/kuberik/pkg/engine/runtime/scheduler/kubernetes"
	"github.com/kuberik/kuberik/pkg/engine/runtime/scheduler/shell"
	corev1 "k8s.io/api/core/v1"
)

var Engine Scheduler

type Scheduler interface {
	Run(name string, namespace corev1.Namespace, exec corev1alpha1.Exec) (io.Reader, chan int, error)
	UpdatePlayPhase(play corev1alpha1.Play, status corev1alpha1.PlayPhaseType) error
	UpdateFrameResult(play corev1alpha1.Play, ID string, result int) error
}

func RunSync(exec corev1alpha1.Exec) ([]byte, error) {
	shell := &shell.Shell{}
	out, result, err := shell.Run("", "", exec)
	if err != nil {
		return []byte(""), err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(out)

	exit := <-result
	if exit != 0 {
		return []byte(""), fmt.Errorf("Exec failed with exit code: %d", exit)
	}
	return buf.Bytes(), nil
}

func InitEngine() {
	Engine = kubernetes.NewKubernetesRuntime(config.Config)
}

func RunAsync(name string, namespace corev1.Namespace, exec corev1alpha1.Exec) (io.Reader, chan int, error) {
	return Engine.Run(name, namespace, exec)
}
