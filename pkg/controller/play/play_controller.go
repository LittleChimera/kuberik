package play

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	kuberikRuntime "github.com/kuberik/kuberik/pkg/engine"
	"github.com/kuberik/kuberik/pkg/engine/scheduler"
	"github.com/kuberik/kuberik/pkg/randutils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_play")

// Add creates a new Play Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePlay{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("play-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Play
	err = c.Watch(&source.Kind{Type: &corev1alpha1.Play{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Play
	err = c.Watch(&source.Kind{Type: &batchv1.Job{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &corev1alpha1.Play{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePlay implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePlay{}

// ReconcilePlay reconciles a Play object
type ReconcilePlay struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Play object and makes changes based on the state read
// and what is in the Play.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePlay) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Play")

	instance := &corev1alpha1.Play{}
	ctx := context.TODO()
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	switch instance.Status.Phase {
	case "", corev1alpha1.PlayPhaseCreated:
		return r.reconcileCreated(instance)
	case corev1alpha1.PlayPhaseInit:
		return r.reconcileInit(instance)
	case corev1alpha1.PlayPhaseRunning:
		return r.reconcileRunning(instance)
	case corev1alpha1.PlayPhaseComplete, corev1alpha1.PlayPhaseFailed, corev1alpha1.PlayPhaseError:
		return r.reconcileComplete(instance)
	}
	return reconcile.Result{}, nil
}

func (r *ReconcilePlay) reconcileCreated(instance *corev1alpha1.Play) (reconcile.Result, error) {
	instance.Status.Phase = corev1alpha1.PlayPhaseInit
	err := r.client.Status().Update(context.TODO(), instance)
	return reconcile.Result{}, err
}

func (r *ReconcilePlay) reconcileInit(instance *corev1alpha1.Play) (reconcile.Result, error) {
	err := func() error {
		err := r.provisionVarsConfigMap(instance)
		if err != nil {
			return err
		}
		err = r.provisionVolumes(instance)
		return err
	}()

	if err != nil {
		instance.Status.Phase = corev1alpha1.PlayPhaseError
		if errUpdate := r.client.Status().Update(context.TODO(), instance); errUpdate != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	instance.Status.Phase = corev1alpha1.PlayPhaseRunning
	err = r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	log.Info(fmt.Sprintf("Running play %s", instance.Name))
	r.populateRandomIDs(&instance.Spec)
	err = r.client.Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcilePlay) reconcileRunning(instance *corev1alpha1.Play) (reconcile.Result, error) {
	if err := r.updateStatus(instance); err != nil {
		return reconcile.Result{}, err
	}

	if instance.Status.Phase == corev1alpha1.PlayPhaseRunning {
		return reconcile.Result{}, kuberikRuntime.PlayNext(instance)
	}
	return reconcile.Result{}, nil
}

func (r *ReconcilePlay) reconcileComplete(instance *corev1alpha1.Play) (reconcile.Result, error) {
	for _, pvcName := range instance.Status.ProvisionedVolumes {
		r.client.Delete(context.TODO(), &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: instance.Namespace,
			},
		})
	}
	instance.Status.ProvisionedVolumes = make(map[string]string)
	err := r.client.Status().Update(context.TODO(), instance)
	log.Info(fmt.Sprintf("Play %s competed with status: %s", instance.Name, instance.Status.Phase))
	return reconcile.Result{}, err
}

func (r *ReconcilePlay) allFrames(playSpec *corev1alpha1.PlaySpec) (frames []*corev1alpha1.Frame) {
	for k := range playSpec.Screenplays {
		for i := range playSpec.Screenplays[k].Scenes {
			for j := range playSpec.Screenplays[k].Scenes[i].Frames {
				frames = append(frames, &(playSpec.Screenplays[k].Scenes[i].Frames[j]))
			}
		}
	}
	return
}

func (r *ReconcilePlay) populateRandomIDs(playSpec *corev1alpha1.PlaySpec) {
	var frames []*corev1alpha1.Frame
	for k := range playSpec.Screenplays {
		for i := range playSpec.Screenplays[k].Scenes {
			for j := range playSpec.Screenplays[k].Scenes[i].Frames {
				frames = append(frames, &(playSpec.Screenplays[k].Scenes[i].Frames[j]))
			}
		}
	}

	randomIDs := randutils.RandList(len(frames))
	for i, f := range frames {
		f.ID = randomIDs[i]
	}
}

func (r *ReconcilePlay) provisionVarsConfigMap(instance *corev1alpha1.Play) error {
	varsConfigMapName := fmt.Sprintf("%s-vars", instance.Name)
	configMapValues := make(map[string]string)
	for _, v := range instance.Spec.Vars {
		configMapValues[v.Name] = v.Value
	}
	varsConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      varsConfigMapName,
			Namespace: instance.Namespace,
		},
		Data: configMapValues,
	}

	err := r.client.Create(context.TODO(), varsConfigMap)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	instance.Status.VarsConfigMap = varsConfigMapName
	return nil
}

func (r *ReconcilePlay) updateStatus(play *corev1alpha1.Play) error {
	jobs := &batchv1.JobList{}
	r.client.List(context.TODO(), jobs, &client.ListOptions{
		LabelSelector: func() labels.Selector {
			ls, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: map[string]string{
					scheduler.JobLabelPlay: play.Name,
				},
			})
			return ls
		}(),
	})

	var updated bool
	for _, j := range jobs.Items {
		frameID := j.Annotations[scheduler.JobAnnotationFrameID]
		if _, ok := play.Status.Frames[frameID]; ok {
			continue
		}

		if play.Status.Frames == nil {
			play.Status.Frames = make(map[string]int)
		}

		updated = true
		if finished, exit := jobResult(&j); finished {
			play.Status.Frames[frameID] = exit
		}
	}

	if !updated {
		return nil
	}

	if len(r.allFrames(&play.Spec)) == len(play.Status.Frames) {
		play.Status.Phase = corev1alpha1.PlayPhaseComplete
	}

	for _, exit := range play.Status.Frames {
		if exit != 0 {
			play.Status.Phase = corev1alpha1.PlayPhaseError
		}
	}

	return r.client.Status().Update(context.TODO(), play)
}

// ProvisionVolumes provisions volumes for the duration of the play
func (r *ReconcilePlay) provisionVolumes(play *corev1alpha1.Play) (err error) {
	if play.Status.ProvisionedVolumes == nil {
		play.Status.ProvisionedVolumes = make(map[string]string)
	}

	for _, volumeClaimTemplate := range play.Spec.VolumeClaimTemplates {
		pvcName := fmt.Sprintf("%s-%s", play.Name, volumeClaimTemplate.Name)

		err = r.client.Create(context.TODO(), &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: play.Namespace,
				Labels: map[string]string{
					"provisionedBy":        "kuberik",
					"core.kuberik.io/play": play.Name,
				},
			},
			Spec: volumeClaimTemplate.Spec,
		})

		if err != nil && !errors.IsAlreadyExists(err) {
			return
		}
		play.Status.ProvisionedVolumes[volumeClaimTemplate.Name] = pvcName
	}
	return
}

func jobResult(job *batchv1.Job) (finished bool, exit int) {
	// Successfully completed a single instance of a job
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed || condition.Type == batchv1.JobComplete {
			finished = true
			if condition.Type == batchv1.JobComplete {
				exit = 0
			} else {
				exit = 1
			}
		}
	}
	return
}
