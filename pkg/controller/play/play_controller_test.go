package play

import (
	"context"
	"fmt"
	"testing"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/scheduler"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	playClient    client.Client
	reconcilePlay *ReconcilePlay
)

func init() {
	play := &corev1alpha1.Play{}
	scheme := scheme.Scheme
	scheme.AddKnownTypes(corev1alpha1.SchemeGroupVersion, play)
	playClient = fake.NewFakeClientWithScheme(scheme)
	reconcilePlay = &ReconcilePlay{client: playClient, scheme: scheme}
	scheduler.InitEngine(scheduler.NewKubernetesScheduler(playClient))
}

func TestPlayCreate(t *testing.T) {
	var (
		name      = "hello-world-run"
		namespace = "default"
	)
	play := &corev1alpha1.Play{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1alpha1.PlaySpec{
			Screenplays: []corev1alpha1.Screenplay{{
				Name: "main",
				Scenes: []corev1alpha1.Scene{{
					Frames: []corev1alpha1.Frame{{
						Name: "test",
					}},
				}},
			}},
		},
	}
	playClient.Create(context.TODO(), play)

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	req := reconcile.Request{
		NamespacedName: nn,
	}
	_, err := reconcilePlay.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// Check the result of reconciliation to make sure it has the desired state.
	playClient.Get(context.TODO(), nn, play)
	if play.Status.Phase != corev1alpha1.PlayPhaseInit {
		t.Error("Reconcile create play didn't reach expected phase")
	}
}

func TestPlayInit(t *testing.T) {
	var (
		name      = "hello-world-init"
		namespace = "default"
	)
	play := &corev1alpha1.Play{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1alpha1.PlaySpec{
			Screenplays: []corev1alpha1.Screenplay{{
				Name: "main",
				Scenes: []corev1alpha1.Scene{{
					Frames: []corev1alpha1.Frame{{
						Name: "test",
						Action: &corev1alpha1.Exec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:    "test",
										Command: []string{"echo", "test"},
										Image:   "alpine",
									}},
								},
							},
						},
					}},
				}},
			}},
		},
		Status: corev1alpha1.PlayStatus{
			Phase: corev1alpha1.PlayPhaseRunning,
		},
	}
	playClient.Create(context.TODO(), play)

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	req := reconcile.Request{
		NamespacedName: nn,
	}
	_, err := reconcilePlay.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// Check the result of reconciliation to make sure it has the desired state.
	playClient.Get(context.TODO(), nn, play)
	if play.Status.Phase != corev1alpha1.PlayPhaseRunning {
		t.Error("Reconcile init play didn't reach expected phase")
	}
}

func TestPlayRunning(t *testing.T) {
	var (
		name      = "hello-world-running"
		namespace = "default"
	)
	play := &corev1alpha1.Play{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1alpha1.PlaySpec{
			Screenplays: []corev1alpha1.Screenplay{{
				Name: "main",
				Scenes: []corev1alpha1.Scene{{
					Frames: []corev1alpha1.Frame{{
						Name: "test",
						Action: &corev1alpha1.Exec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:    "test",
										Command: []string{"echo", "test"},
										Image:   "alpine",
									}},
								},
							},
						},
					}},
				}},
			}},
		},
		Status: corev1alpha1.PlayStatus{
			Phase: corev1alpha1.PlayPhaseRunning,
		},
	}
	playClient.Create(context.TODO(), play)

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	req := reconcile.Request{
		NamespacedName: nn,
	}
	_, err := reconcilePlay.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// Check the result of reconciliation to make sure it has the desired state.
	playClient.Get(context.TODO(), nn, play)
	if play.Status.Phase != corev1alpha1.PlayPhaseRunning {
		t.Error("Initialize play didn't reach expected phase")
	}

	job := &batchv1.Job{}
	err = playClient.Get(context.TODO(), types.NamespacedName{
		Name:      fmt.Sprintf("%.46s-%.16s", play.Name, play.Screenplay("main").Scenes[0].Frames[0].ID),
		Namespace: play.Namespace,
	}, job)
	if err != nil {
		t.Error("Expected a created job to run the play")
	}

	job.Status.Conditions = append(job.Status.Conditions, batchv1.JobCondition{
		Type: batchv1.JobComplete,
	})
	playClient.Status().Update(context.TODO(), job)
	_, err = reconcilePlay.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	playClient.Get(context.TODO(), nn, play)
	if play.Status.Phase != corev1alpha1.PlayPhaseComplete {
		t.Error("Play didn't finish succesfully")
	}
}
