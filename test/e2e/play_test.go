package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kuberik/kuberik/pkg/apis"
	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"

	"github.com/kuberik/kuberik/test/e2e/e2eutil"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	operatore2eutil "github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestPlay(t *testing.T) {
	playList := &corev1alpha1.PlayList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, playList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("play-group", func(t *testing.T) {
		t.Run("Cluster", HelloWorldPlay)
	})
}

func playRunTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	helloWorldAction := &corev1alpha1.Exec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:    "hello",
					Image:   "alpine",
					Command: []string{"echo", "Hello world!"},
				}},
			},
		},
	}

	playName := "hello-world"
	helloWorldPlay := &corev1alpha1.Play{
		ObjectMeta: metav1.ObjectMeta{
			Name:      playName,
			Namespace: namespace,
		},
		Spec: corev1alpha1.PlaySpec{
			Screenplays: []corev1alpha1.Screenplay{{
				Name: "main",
				Scenes: []corev1alpha1.Scene{
					{
						Name: "first-scene",
						Frames: []corev1alpha1.Frame{
							{
								Name:   "first-hello-a",
								Action: helloWorldAction,
							},
							{
								Name:   "first-hello-b",
								Action: helloWorldAction,
							},
						},
					},
					{
						Name: "second-scene",
						Frames: []corev1alpha1.Frame{
							{
								Name:   "second-hello-a",
								Action: helloWorldAction,
							},
							{
								Name:   "second-hello-b",
								Action: helloWorldAction,
							},
						},
					},
				},
			}},
		},
	}
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(context.TODO(), helloWorldPlay, &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       cleanupTimeout,
		RetryInterval: cleanupRetryInterval,
	})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForPlayFinished(t, f.Client.Client, namespace, playName, retryInterval, timeout)
	if err != nil {
		return err
	}

	finishedPlay := &corev1alpha1.Play{}
	err = f.Client.Client.Get(context.TODO(), types.NamespacedName{Name: playName, Namespace: namespace}, finishedPlay)
	if err != nil {
		return err
	}

	if finishedPlay.Status.Phase != corev1alpha1.PlayPhaseComplete {
		t.Errorf("Expected play %s to finish with %s, but instead resulted with %s.\n", playName, corev1alpha1.PlayPhaseComplete, finishedPlay.Status.Phase)
	}
	return nil
}

func HelloWorldPlay(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	err = operatore2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "kuberik", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = playRunTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
