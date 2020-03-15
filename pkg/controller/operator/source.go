package operator

import (
	"fmt"

	"github.com/kuberik/kuberik/pkg/controller/operator/internal"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// var log = logf.KBLog.WithName("source")

// Kind is used to provide a source of events originating inside the cluster from Watches (e.g. Pod Create)
type Kind struct {
	// Type is the type of object to watch.  e.g. &v1.Pod{}

	schema.GroupVersionKind

	// cache used to watch APIs
	cache cache.Cache
}

var _ source.Source = &Kind{}

// Start is internal and should be called only by the Controller to register an EventHandler with the Informer
// to enqueue reconcile.Requests.
func (ks *Kind) Start(handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	prct ...predicate.Predicate) error {

	// Type should have been specified by the user.
	// if ks.Type == nil {
	// 	return fmt.Errorf("must specify Kind.Type")
	// }

	// cache should have been injected before Start was called
	if ks.cache == nil {
		return fmt.Errorf("must call CacheInto on Kind before calling Start")
	}

	// Lookup the Informer from the Cache and add an EventHandler which populates the Queue
	log.Info("Informing for ", ks.GroupVersionKind)
	i, err := ks.cache.GetInformerForKind(ks.GroupVersionKind)
	if err != nil {
		if kindMatchErr, ok := err.(*meta.NoKindMatchError); ok {
			log.Error(err, "if kind is a CRD, it should be installed before calling Start",
				"kind", kindMatchErr.GroupKind)
		}
		return err
	}
	i.AddEventHandler(internal.EventHandler{Queue: queue, EventHandler: handler, Predicates: prct})
	return nil
}

func (ks *Kind) String() string {
	if !ks.GroupVersionKind.Empty() {
		return fmt.Sprintf("kind source: %v", ks.GroupVersionKind.String())
	}
	return fmt.Sprintf("kind source: unknown GVK")
}

var _ inject.Cache = &Kind{}

// InjectCache is internal should be called only by the Controller.  InjectCache is used to inject
// the Cache dependency initialized by the ControllerManager.
func (ks *Kind) InjectCache(c cache.Cache) error {
	if ks.cache == nil {
		ks.cache = c
	}
	return nil
}
