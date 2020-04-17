package e2eutil

import (
	"context"
	"testing"
	"time"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitForPlayFinished checks to see if a given play finished after a specified
// amount of time. If the play does not finishes after 5 * retries seconds,
// the function returns an error.
func WaitForPlayFinished(t *testing.T, client client.Client, namespace, name string,
	retryInterval, timeout time.Duration) error {
	return waitForPlayFinished(t, client, namespace, name, retryInterval, timeout)
}

func waitForPlayFinished(t *testing.T, client client.Client, namespace, name string,
	retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		play := &corev1alpha1.Play{}
		err = client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, play)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s play\n", name)
				return false, nil
			}
			return false, err
		}

		switch play.Status.Phase {
		case corev1alpha1.PlayPhaseComplete,
			corev1alpha1.PlayPhaseFailed,
			corev1alpha1.PlayPhaseError:
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Play %s finished\n", name)
	return nil
}
