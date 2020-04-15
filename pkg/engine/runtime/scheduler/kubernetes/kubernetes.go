package kubernetes

import (
	"fmt"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
)

const (
	// maximum length of job name
	maxJobNameLength = 63
)

const (
	// JobLabelPlay is name of a label which stores name of the play that owns frame of this job
	JobLabelPlay = "kuberik.io/play"
	// JobLabelFrameID is name of a label which stores ID of the frame that owns the job
	JobLabelFrameID = "kuberik.io/frameID"
)

var updateLock sync.Mutex

// Scheduler defines a Scheduler which executes Plays on Kubernetes
type Scheduler struct {
	config *rest.Config
	client *kubernetes.Clientset
}

// NewScheduler create a new NewScheduler
func NewScheduler(c *rest.Config) *Scheduler {
	client, _ := kubernetes.NewForConfig(c)

	return &Scheduler{
		config: c,
		client: client,
	}
}

// Run creates an execution on Kubernetes
func (r *Scheduler) Run(play *corev1alpha1.Play, frameID string) error {
	jobDefinition := newRunJob(play, frameID)
	// Try to recover first
	_, err := r.client.BatchV1().Jobs(play.Namespace).Get(jobDefinition.GetName(), metav1.GetOptions{})
	if err != nil {
		_, err = r.client.BatchV1().Jobs(play.Namespace).Create(jobDefinition)
		return err
	}

	return nil
}

var (
	falseVal       = false
	trueVal        = true
	zero     int32 = 0
)

func newRunJob(play *corev1alpha1.Play, frameID string) *batchv1.Job {
	e := play.Frame(frameID).Action
	labels := map[string]string{
		JobLabelPlay:    play.Name,
		JobLabelFrameID: frameID,
	}
	if e.BackoffLimit == nil {
		e.BackoffLimit = &zero
	}
	if e.Template.Spec.RestartPolicy == "" {
		e.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			// maximum string for job name is 63 characters.
			Name:   fmt.Sprintf("%.46s-%.16s", play.Name, frameID),
			Labels: labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: play.APIVersion,
				Kind:       play.Kind,
				Name:       play.Name,
				UID:        play.UID,
				Controller: &trueVal,
			}},
		},
		Spec: *e,
	}

	return job
}

func (r *Scheduler) getJobWatcher(job *batchv1.Job) watch.Interface {
	watcher, _ := r.client.BatchV1().Jobs(job.Namespace).Watch(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", job.Name),
	})
	return watcher
}

func (r *Scheduler) watchJob(result chan int, jobDefinition *batchv1.Job) {
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
