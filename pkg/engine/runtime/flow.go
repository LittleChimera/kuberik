package runtime

import (
	"bufio"
	"fmt"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/tidwall/gjson"

	"github.com/kuberik/kuberik/cmd/kuberik/cmd"
	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/runtime/scheduler"
	"github.com/kuberik/kuberik/pkg/kubeutils"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	frameCopyIndexVar = "FRAME_COPY_INDEX"
)

func Play(livePlay corev1alpha1.Play) error {
	populateVars(&livePlay.Spec.Screenplay, livePlay.Status.VarsConfigMap)
	expandCopies(&livePlay.Spec.Screenplay)
	expandProvisionedVolumes(&livePlay)
	go func() {
		success := true
		for i, _ := range livePlay.Spec.Screenplay.Scenes {
			success = success && playScene(livePlay, &livePlay.Spec.Screenplay.Scenes[i])
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

// TODO remove
func CreatePlay(movie *corev1alpha1.Movie, payload string) error {
	resolveValues := func(vars corev1alpha1.Vars, payload string) (values []corev1alpha1.Var, err error) {
		for _, v := range vars {
			value := v.Value
			if v.ValueFrom != nil {
				if v.ValueFrom.InputRef != nil {
					result := gjson.Get(payload, v.ValueFrom.InputRef.GJSONPath)
					if result.Exists() {
						value = result.String()
					} else {
						return nil, err
					}
				} else if v.ValueFrom.ConfigMapKeyRef != nil {
					value = ""
				} else if v.ValueFrom.SecretKeyRef != nil {
					value = ""
				}
			}
			values = append(values, corev1alpha1.Var{
				Name:  v.Name,
				Value: value,
			})
		}
		return values, err
	}
	resolvedVars, _ := resolveValues(movie.Spec.Screenplay.Vars, payload)
	play, err := cmd.PlayFromMovie(movie, resolvedVars)
	if err != nil {
		log.Error(err)
		return err
	}
	if _, err = cmd.CreatePlayInstance(play); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func playScene(livePlay corev1alpha1.Play, scene *corev1alpha1.Scene) bool {
	if ok := initializeScene(livePlay, scene.Name); !ok {
		log.Infof("Skipping scene: %s", scene.Name)
		return true
	}

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
	output, result, err := scheduler.RunAsync(executionName, kubeutils.NamespaceObject(livePlay.Namespace), frame.Action)
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

func initializeScene(livePlay corev1alpha1.Play, sceneName string) bool {
	// TODO check if scene was already started
	liveScene, _ := livePlay.Spec.Screenplay.Scene(sceneName)
	if len(liveScene.When) > 0 {
		return liveScene.When.Evaluate(livePlay.Spec.Screenplay.Vars)
	}
	return true
}

func finalizeScene(livePlay corev1alpha1.Play, sceneName string, exit int) {
	liveScene, _ := livePlay.Spec.Screenplay.Scene(sceneName)
	if len(liveScene.Pass) > 0 {
		if ok := liveScene.Pass.Evaluate(livePlay.Spec.Screenplay.Vars); !ok {
			return
		}
	}

	if exit != 0 {
		log.Errorf("Scene failed: %v", liveScene.Name)
		scheduler.Engine.UpdatePlayPhase(livePlay, corev1alpha1.PlayFailed)
	}

	// Scene failed so don't proceed onto the next one.
	if exit != 0 {
		return
	}
}

func expandCopies(screenplay *corev1alpha1.Screenplay) {
	for si := range screenplay.Scenes {
		var frames []corev1alpha1.Frame
		for _, f := range screenplay.Scenes[si].Frames {
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
		screenplay.Scenes[si].Frames = frames
	}
}

func expandProvisionedVolumes(play *corev1alpha1.Play) {
	screenplay := play.Spec.Screenplay
	volumes := play.Status.ProvisionedVolumes
	for si := range screenplay.Scenes {
		for fi := range screenplay.Scenes[si].Frames {
		volumes:
			for volumeName, provisionedVolumeName := range volumes {
				// TODO expand logic for initContainers as well
				for _, container := range screenplay.Scenes[si].Frames[fi].Action.Template.Spec.Containers {
					for _, m := range container.VolumeMounts {
						if m.Name == volumeName {
							screenplay.Scenes[si].Frames[fi].Action.Template.Spec.Volumes = append(
								screenplay.Scenes[si].Frames[fi].Action.Template.Spec.Volumes,
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

func populateVars(screenplay *corev1alpha1.Screenplay, varsConfigMap string) {
	if varsConfigMap == "" {
		return
	}
	mountName := "kuberik-vars"
	mountPath := "/kuberik/vars"
	for i := range screenplay.Scenes {
		for j := range screenplay.Scenes[i].Frames {
			screenplay.Scenes[i].Frames[j].Action.Template.Spec.Volumes = append(
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
			for ci := range screenplay.Scenes[i].Frames[j].Action.Template.Spec.Containers {
				screenplay.Scenes[i].Frames[j].Action.Template.Spec.Containers[ci].EnvFrom = append(
					screenplay.Scenes[i].Frames[j].Action.Template.Spec.Containers[ci].EnvFrom,
					corev1.EnvFromSource{
						ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: varsConfigMap,
							},
						},
					},
				)
				screenplay.Scenes[i].Frames[j].Action.Template.Spec.Containers[ci].VolumeMounts = append(
					screenplay.Scenes[i].Frames[j].Action.Template.Spec.Containers[ci].VolumeMounts,
					corev1.VolumeMount{
						Name:      mountName,
						MountPath: mountPath,
					},
				)
			}
			for ci := range screenplay.Scenes[i].Frames[j].Action.Template.Spec.InitContainers {
				screenplay.Scenes[i].Frames[j].Action.Template.Spec.InitContainers[ci].EnvFrom = append(
					screenplay.Scenes[i].Frames[j].Action.Template.Spec.InitContainers[ci].EnvFrom,
					corev1.EnvFromSource{
						ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: varsConfigMap,
							},
						},
					},
				)
				screenplay.Scenes[i].Frames[j].Action.Template.Spec.InitContainers[ci].VolumeMounts = append(
					screenplay.Scenes[i].Frames[j].Action.Template.Spec.InitContainers[ci].VolumeMounts,
					corev1.VolumeMount{
						Name:      mountName,
						MountPath: mountPath,
					},
				)
			}
		}
	}
}

func LoadScreenplay(movie *corev1alpha1.MovieSpec) (*corev1alpha1.Screenplay, error) {
	return movie.Screenplay, nil
}
