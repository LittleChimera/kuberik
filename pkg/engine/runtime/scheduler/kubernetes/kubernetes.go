package kubernetes

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
)

const (
	// maximum length of job name
	maxJobNameLength = 63
)

var updateLock sync.Mutex

// KubernetesRuntime defines a Scheduler which executes Plays on Kubernetes
type KubernetesRuntime struct {
	config *rest.Config
	client *kubernetes.Clientset
}

// NewKubernetesRuntime create a new NewKubernetesRuntime
func NewKubernetesRuntime(c *rest.Config) *KubernetesRuntime {
	client, _ := kubernetes.NewForConfig(c)

	return &KubernetesRuntime{
		config: c,
		client: client,
	}
}

// Run creates an execution on Kubernetes
func (r *KubernetesRuntime) Run(play *corev1alpha1.Play, frameID string) (chan int, error) {
	result := make(chan int)

	jobDefinition := newRunJob(play, frameID)
	// Try to recover first
	jobInstance, err := r.client.BatchV1().Jobs(play.Namespace).Get(jobDefinition.GetName(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		jobInstance, err = r.client.BatchV1().Jobs(play.Namespace).Create(jobDefinition)
	}
	// TODO retry
	if err != nil {
		return nil, err
	}

	go r.watchJob(result, jobInstance)

	return result, nil
}

var (
	falseVal       = false
	zero     int32 = 0
)

func newRunJob(play *corev1alpha1.Play, frameID string) *batchv1.Job {
	e := play.Frame(frameID).Action
	labels := map[string]string{
		"runner": "kuberik",
	}
	if e.BackoffLimit == nil {
		e.BackoffLimit = &zero
	}
	if e.Template.Spec.RestartPolicy == "" {
		e.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	}

	t := true
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			// maximum string for job name is 63 characters.
			Name:   fmt.Sprintf("%.46s-%.16s", play.Name, frameID),
			Labels: labels,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: play.APIVersion,
					Kind:       play.Kind,
					Name:       play.Name,
					UID:        play.UID,
					Controller: &t,
				},
			},
		},
		Spec: *e,
	}

	return job
}

func (r *KubernetesRuntime) getJobWatcher(job *batchv1.Job) watch.Interface {
	watcher, _ := r.client.BatchV1().Jobs(job.Namespace).Watch(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", job.Name),
	})
	return watcher
}

func (r *KubernetesRuntime) watchJob(result chan int, jobDefinition *batchv1.Job) {
	finish := func(job *batchv1.Job) bool {
		// Successfully completed a single instance of a job
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobFailed || condition.Type == batchv1.JobComplete {
				// Exit code is N where N is the number of Pods that failed. If the job
				// ran to completion, the exit code will be 0.
				if condition.Type == batchv1.JobComplete {
					result <- 0
				} else {
					result <- 1
				}
				return true
			}
		}
		return false
	}

	watcher := r.getJobWatcher(jobDefinition)
	defer watcher.Stop()
	results := watcher.ResultChan()

	currentJob, _ := r.client.BatchV1().Jobs(jobDefinition.Namespace).Get(jobDefinition.GetName(), metav1.GetOptions{})
	if finish(currentJob) {
		return
	}

	for event := range results {
		job, _ := event.Object.(*batchv1.Job)
		log.Infof("Job: %s active: %d, succeeded: %d, failed: %d", job.Name, job.Status.Active, job.Status.Succeeded, job.Status.Failed)
		if finish(job) {
			log.Infof("Finished job watcher for %s", job.Name)
			return
		}
	}
}

func (r *KubernetesRuntime) updateStatus(play corev1alpha1.Play, transform func(*corev1alpha1.Play)) error {
	// Accessing frame status map is unsafe
	updateLock.Lock()
	defer updateLock.Unlock()
	instance := corev1alpha1.Play{}
	config.Client.Get(context.TODO(), types.NamespacedName{Namespace: play.Namespace, Name: play.Name}, &instance)
	transform(&instance)
	return config.Client.Status().Update(context.TODO(), &instance)
}

// UpdatePlayPhase updates the phase of a Play
func (r *KubernetesRuntime) UpdatePlayPhase(play corev1alpha1.Play, phase corev1alpha1.PlayPhaseType) error {
	return r.updateStatus(play, func(instance *corev1alpha1.Play) {
		instance.Status.Phase = phase
	})
}

// UpdateFrameResult updates the results of a Frame in the Play
func (r *KubernetesRuntime) UpdateFrameResult(play corev1alpha1.Play, ID string, result int) error {
	return r.updateStatus(play, func(instance *corev1alpha1.Play) {
		if instance.Status.Frames == nil {
			instance.Status.Frames = make(map[string]int)
		}
		instance.Status.Frames[ID] = result
	})
}
