package operator

import (
	"fmt"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("controller_operator")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Operator Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, movie *corev1alpha1.MovieSpec, gvk schema.GroupVersionKind) error {
	resource := newResourceFromGVK(gvk)
	register(mgr, resource)
	return add(mgr, newReconciler(mgr, movie, resource), resource)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, movie *corev1alpha1.MovieSpec, resource Resource) reconcile.Reconciler {
	return &ReconcileOperator{config: mgr.GetConfig(), movie: movie, scheme: mgr.GetScheme(), resource: resource}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, resource Resource) error {
	// Create a new controller
	c, err := controller.New("operator-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&Kind{GroupVersionKind: resource.GroupVersionKind}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileOperator implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOperator{}

// ReconcileOperator reconciles a Operator object
type ReconcileOperator struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	config   *rest.Config
	scheme   *runtime.Scheme
	resource Resource
	movie    *corev1alpha1.MovieSpec
}

// Reconcile reads that state of the cluster for a Operator object and makes changes based on the state read
// and what is in the Operator.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOperator) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Operator")

	client, _ := dynamic.NewForConfig(r.config)
	object, _ := client.
		Resource(schema.GroupVersionResource{
			Group:    r.resource.Group,
			Version:  r.resource.Version,
			Resource: r.resource.plural(),
		}).Namespace(request.Namespace).Get(request.Name, metav1.GetOptions{})

	if r.movie.Screenplay == nil {
		return reconcile.Result{}, fmt.Errorf("Screenplay not defined for movie: %s", r.movie.Name)
	}

	// objectJSON, err := object.MarshalJSON()
	_, err := object.MarshalJSON()
	if err != nil {
		return reconcile.Result{}, err
	}
	// return reconcile.Result{}, kuberikruntime.InitPlay(string(objectJSON), *r.movie.Screenplay, "reconcile")
	return reconcile.Result{}, nil
}
