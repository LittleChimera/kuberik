package runtime

import (
	"fmt"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/runtime/scheduler"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	frameCopyIndexVar  = "FRAME_COPY_INDEX"
	mainScreenplayName = "main"
)

// PlayNext executes all frames that are possible to play at the current stage
func PlayNext(play *corev1alpha1.Play) error {
	// Expand definition
	populateVars(&play.Spec, play.Status.VarsConfigMap)
	expandCopies(&play.Spec)
	expandProvisionedVolumes(play)
	return playScreenplay(play, mainScreenplayName)
}

func playScreenplay(play *corev1alpha1.Play, name string) error {
	screenplay := play.Screenplay(name)
	if screenplay == nil {
		return fmt.Errorf("Play doesn't have a main screenplay")
	}

	for si := range screenplay.Scenes {
		sceneFinished := true
		for _, frame := range screenplay.Scenes[si].Frames {
			_, ok := play.Status.Frames[frame.ID]
			sceneFinished = sceneFinished && ok
		}

		if sceneFinished {
			continue
		}

		return playScene(play, &screenplay.Scenes[si])
	}

	return corev1alpha1.NewError(corev1alpha1.NoMoreFrames)
}

func playScene(play *corev1alpha1.Play, scene *corev1alpha1.Scene) error {
	// var exit int
	for _, frame := range scene.Frames {
		if _, ok := play.Status.Frames[frame.ID]; ok {
			continue
		}
		err := playFrame(play, frame.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func playFrame(play *corev1alpha1.Play, frameID string) error {
	err := scheduler.Engine.Run(play, frameID)
	if err != nil {
		log.Errorf("Failed to play %s from %s: %s", frameID, play.Name, err)
	}
	return err
}

func expandCopies(playSpec *corev1alpha1.PlaySpec) {
	for k := range playSpec.Screenplays {
		for si := range playSpec.Screenplays[k].Scenes {
			var frames []corev1alpha1.Frame
			for _, f := range playSpec.Screenplays[k].Scenes[si].Frames {
				if f.Copies > 1 {
					for i := 0; i < f.Copies; i++ {
						fc := f.Copy()

						fc.ID = fmt.Sprintf("%s-%v", fc.ID, i)
						fc.Name = fmt.Sprintf("%s-%v", fc.Name, i)
						for ci := range fc.Action.Template.Spec.Containers {
							fc.Action.Template.Spec.Containers[ci].Env = append(fc.Action.Template.Spec.Containers[ci].Env, corev1.EnvVar{
								Name:  frameCopyIndexVar,
								Value: fmt.Sprintf("%v", i),
							})
						}
						frames = append(frames, fc)
					}
				} else {
					frames = append(frames, f)
				}
			}
			playSpec.Screenplays[k].Scenes[si].Frames = frames
		}
	}
}

func expandProvisionedVolumes(play *corev1alpha1.Play) {
	// screenplay := play.Spec.Screenplay
	volumes := play.Status.ProvisionedVolumes
	for k := range play.Spec.Screenplays {
		for si := range play.Spec.Screenplays[k].Scenes {
			for fi := range play.Spec.Screenplays[k].Scenes[si].Frames {
			volumes:
				for volumeName, provisionedVolumeName := range volumes {
					for _, container := range play.Spec.Screenplays[k].Scenes[si].Frames[fi].Action.Template.Spec.Containers {
						for _, m := range container.VolumeMounts {
							if m.Name == volumeName {
								play.Spec.Screenplays[k].Scenes[si].Frames[fi].Action.Template.Spec.Volumes = append(
									play.Spec.Screenplays[k].Scenes[si].Frames[fi].Action.Template.Spec.Volumes,
									corev1.Volume{
										Name: volumeName,
										VolumeSource: corev1.VolumeSource{
											PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
												ClaimName: provisionedVolumeName,
											},
										},
									},
								)
								continue volumes
							}
						}
					}
					for _, container := range play.Spec.Screenplays[k].Scenes[si].Frames[fi].Action.Template.Spec.InitContainers {
						for _, m := range container.VolumeMounts {
							if m.Name == volumeName {
								play.Spec.Screenplays[k].Scenes[si].Frames[fi].Action.Template.Spec.Volumes = append(
									play.Spec.Screenplays[k].Scenes[si].Frames[fi].Action.Template.Spec.Volumes,
									corev1.Volume{
										Name: volumeName,
										VolumeSource: corev1.VolumeSource{
											PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
												ClaimName: provisionedVolumeName,
											},
										},
									},
								)
								continue volumes
							}
						}
					}
				}
			}
		}
	}
}

func populateVars(playSpec *corev1alpha1.PlaySpec, varsConfigMap string) {
	if varsConfigMap == "" {
		return
	}
	mountName := "kuberik-vars"
	mountPath := "/kuberik/vars"
	for k, screenplay := range playSpec.Screenplays {
		for i, scene := range screenplay.Scenes {
			for j, frame := range scene.Frames {
				playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.Volumes = append(
					screenplay.Scenes[i].Frames[j].Action.Template.Spec.Volumes,
					corev1.Volume{
						Name: mountName,
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: varsConfigMap,
								},
							},
						},
					},
				)
				for ci := range frame.Action.Template.Spec.Containers {
					playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.Containers[ci].EnvFrom = append(
						playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.Containers[ci].EnvFrom,
						corev1.EnvFromSource{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: varsConfigMap,
								},
							},
						},
					)
					playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.Containers[ci].VolumeMounts = append(
						playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.Containers[ci].VolumeMounts,
						corev1.VolumeMount{
							Name:      mountName,
							MountPath: mountPath,
						},
					)
				}
				for ci := range playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.InitContainers {
					playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.InitContainers[ci].EnvFrom = append(
						playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.InitContainers[ci].EnvFrom,
						corev1.EnvFromSource{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: varsConfigMap,
								},
							},
						},
					)
					playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.InitContainers[ci].VolumeMounts = append(
						playSpec.Screenplays[k].Scenes[i].Frames[j].Action.Template.Spec.InitContainers[ci].VolumeMounts,
						corev1.VolumeMount{
							Name:      mountName,
							MountPath: mountPath,
						},
					)
				}
			}
		}
	}
}
