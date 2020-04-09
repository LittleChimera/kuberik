package runtime

import (
	"bufio"
	"fmt"

	_ "github.com/jinzhu/gorm/dialects/sqlite"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/runtime/scheduler"
	"github.com/kuberik/kuberik/pkg/kubeutils"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	frameCopyIndexVar  = "FRAME_COPY_INDEX"
	mainScreenplayName = "main"
)

func Play(livePlay corev1alpha1.Play) error {
	var mainPlay *corev1alpha1.Screenplay
	for i := range livePlay.Spec.Screenplays {
		if livePlay.Spec.Screenplays[i].Name == mainScreenplayName {
			mainPlay = &livePlay.Spec.Screenplays[i]
		}
	}
	if mainPlay == nil {
		return fmt.Errorf("Play doesn't have a main screenplay")
	}
	populateVars(&livePlay.Spec, livePlay.Status.VarsConfigMap)
	expandCopies(&livePlay.Spec)
	expandProvisionedVolumes(&livePlay)
	go func() {
		success := true
		for i, _ := range mainPlay.Scenes {
			success = success && playScene(livePlay, &mainPlay.Scenes[i])
			if !success {
				break
			}
		}

		var playEnd corev1alpha1.PlayPhaseType
		if success {
			playEnd = corev1alpha1.PlayComplete
		} else {
			playEnd = corev1alpha1.PlayFailed
		}
		scheduler.Engine.UpdatePlayPhase(livePlay, playEnd)
	}()
	return nil
}

func playScene(livePlay corev1alpha1.Play, scene *corev1alpha1.Scene) bool {
	// var exit int
	exits := make(chan int)
	for i, _ := range scene.Frames {
		frame := scene.Frames[i]
		go func() {
			exit, _ := playFrame(livePlay, frame)
			err := scheduler.Engine.UpdateFrameResult(livePlay, frame.ID, exit)
			if err != nil {
				log.Warn(fmt.Errorf("Updating frame result failed: %s", err))
			}
			exits <- exit
		}()
	}

	exitTotal := 0
	for _ = range scene.Frames {
		exitTotal = <-exits | exitTotal
	}

	finalizeScene(livePlay, scene.Name, exitTotal)
	if scene.IgnoreErrors {
		return true
	}
	return exitTotal == 0
}

func playFrame(livePlay corev1alpha1.Play, frame corev1alpha1.Frame) (int, error) {
	if exit, recovered := livePlay.Status.Frames[frame.ID]; recovered {
		return exit, nil
	}

	// maximum string for job name is 63 characters.
	executionName := fmt.Sprintf("%.29s-%.16s-%.16s", livePlay.Name, frame.Name, frame.ID)
	output, result, err := scheduler.RunAsync(executionName, kubeutils.NamespaceObject(livePlay.Namespace), *frame.Action)
	if err != nil {
		log.Errorf("Failed to play frame (%s): %s", frame.Name, err)
		scheduler.Engine.UpdatePlayPhase(livePlay, corev1alpha1.PlayError)
		return 1, err
	}
	buffer := bufio.NewReaderSize(output, 32*1024)
	for {
		line, _, err := buffer.ReadLine()

		if err != nil {
			break
		}
		log.Infof("Task %s: %s", frame.Name, line)
	}
	exit := <-result
	if exit != 0 && frame.IgnoreErrors {
		exit = 0
	}
	return exit, nil
}

func finalizeScene(livePlay corev1alpha1.Play, sceneName string, exit int) {
	if exit != 0 {
		scheduler.Engine.UpdatePlayPhase(livePlay, corev1alpha1.PlayFailed)
	}

	// Scene failed so don't proceed onto the next one.
	if exit != 0 {
		return
	}
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
					// TODO expand logic for initContainers as well
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
