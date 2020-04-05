package play

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/kuberik/kuberik/pkg/engine/config"
	kuberikRuntime "github.com/kuberik/kuberik/pkg/engine/runtime"
	"github.com/kuberik/kuberik/pkg/randutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_play")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

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

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Play
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
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
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePlay) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Play")

	// Fetch the Play instance
	instance := &corev1alpha1.Play{}
	ctx := context.TODO()
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	switch instance.Status.Phase {
	case "":
		err := func() error {
			err := provisionVarsConfigMap(instance)
			if err != nil {
				return err
			}
			err = provisionVolumes(instance)
			return err
		}()

		if err != nil {
			instance.Status.Phase = corev1alpha1.PlayError
			if errUpdate := r.client.Status().Update(ctx, instance); errUpdate != nil {
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{}, err
		}

		instance.Status.Phase = corev1alpha1.PlayCreated
		instance.Status.Runner = config.RunnerID
		err = r.client.Status().Update(ctx, instance)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}

	case corev1alpha1.PlayCreated:
		instance.Status.Phase = corev1alpha1.PlayRunning
		instance.Status.Runner = config.RunnerID
		err := r.client.Status().Update(ctx, instance)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}

		varsConfigMap := corev1.ConfigMap{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Status.VarsConfigMap, Namespace: instance.Namespace}, &varsConfigMap)
		if err != nil {
			return reconcile.Result{}, err
		}

		// TODO populate configMapRef instead of value to be able to retrieve vars dynamically
		for vi := range instance.Spec.Vars {
			for k, v := range varsConfigMap.Data {
				if k == instance.Spec.Vars[vi].Name {
					instance.Spec.Vars[vi].Value = v
				}
			}
		}

		log.Info(fmt.Sprintf("Running play %s", instance.Name))
		populateRandomIDs(&instance.Spec)
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		// TODO r.client.Get(ctx, request.NamespacedName, instance)
		kuberikRuntime.Play(*instance)
	case corev1alpha1.PlayRunning:
		if instance.Status.Runner != config.RunnerID {
			log.Info(fmt.Sprintf("Recovering %s/%s...", instance.Namespace, instance.Name))
			instance.Status.Runner = config.RunnerID
			err := r.client.Status().Update(ctx, instance)
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			// TODO r.client.Get(ctx, request.NamespacedName, instance)
			kuberikRuntime.Play(*instance)
		}
	case corev1alpha1.PlayComplete, corev1alpha1.PlayFailed, corev1alpha1.PlayError:
		for _, pvcName := range instance.Status.ProvisionedVolumes {
			r.client.Delete(context.TODO(), &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: instance.Namespace,
				},
			})
		}
		instance.Status.ProvisionedVolumes = make(map[string]string)
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}
	return reconcile.Result{}, nil
}

func populateRandomIDs(playSpec *corev1alpha1.PlaySpec) {
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

func provisionVarsConfigMap(instance *corev1alpha1.Play) error {
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

	err := config.Client.Create(context.TODO(), varsConfigMap)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	instance.Status.VarsConfigMap = varsConfigMapName
	return nil
}

// ProvisionVolumes provisions volumes for the duration of the play
func provisionVolumes(play *corev1alpha1.Play) (err error) {
	if play.Status.ProvisionedVolumes == nil {
		play.Status.ProvisionedVolumes = make(map[string]string)
	}

	for _, volumeClaimTemplate := range play.Spec.VolumeClaimTemplates {
		pvcName := fmt.Sprintf("%s-%s", play.Name, volumeClaimTemplate.Name)

		err = config.Client.Create(context.TODO(), &corev1.PersistentVolumeClaim{
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
