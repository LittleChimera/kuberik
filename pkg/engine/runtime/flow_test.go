package runtime

import (
	"testing"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestExpandLoops(t *testing.T) {
	screenplay := corev1alpha1.Screenplay{
		Scenes: []corev1alpha1.Scene{
			corev1alpha1.Scene{
				Frames: []corev1alpha1.Frame{
					corev1alpha1.Frame{
						Copies: 3,
						Action: &corev1alpha1.Exec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	expandCopies(&corev1alpha1.PlaySpec{
		Screenplays: []corev1alpha1.Screenplay{
			screenplay,
		},
	})

	if len(screenplay.Scenes[0].Frames) != 3 {
		t.Errorf("Expand loop doesn't add new frames")
	}
	found := false
	for _, e := range screenplay.Scenes[0].Frames[1].Action.Template.Spec.Containers[0].Env {
		if e.Name == frameCopyIndexVar {
			found = true
			if e.Value != "1" {
				t.Errorf("Index variable is not correctly populated")
			}
		}
	}
	if !found {
		t.Errorf("Index variable is not injected")
	}
}
