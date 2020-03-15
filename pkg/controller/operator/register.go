package operator

import (
	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

func register(mgr manager.Manager, resource Resource) {
	registerCRD(mgr.GetConfig(), resource)
	registerScheme(mgr.GetScheme(), resource)
}

func registerCRD(config *rest.Config, resource Resource) {
	client, err := apiextension.NewForConfig(config)
	if err != nil {
		// TODO
	}
	_, err = client.ApiextensionsV1beta1().CustomResourceDefinitions().Create(&apiextensionv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: resource.crdName(),
		},
		Spec: apiextensionv1beta1.CustomResourceDefinitionSpec{
			Group:   resource.Group,
			Version: resource.Version,
			Scope:   apiextensionv1beta1.NamespaceScoped,
			Names: apiextensionv1beta1.CustomResourceDefinitionNames{
				Kind:     resource.Kind,
				ListKind: resource.listKind(),
				Plural:   resource.plural(),
				Singular: resource.singular(),
			},
		},
	})
	if err != nil && apierrors.IsAlreadyExists(err) {
		log.Error(err, "Failed to create CRD")
	}
}

func registerScheme(runtimeScheme *runtime.Scheme, resource Resource) {
	SchemeGroupVersion := schema.GroupVersion{Group: resource.Group, Version: resource.Version}
	SchemeBuilder := &scheme.Builder{GroupVersion: SchemeGroupVersion}
	SchemeBuilder.SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   resource.Group,
			Version: resource.Version,
			Kind:    resource.Kind,
		}, &unstructured.Unstructured{})
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   resource.Group,
			Version: resource.Version,
			Kind:    resource.listKind(),
		}, &unstructured.Unstructured{})
		metav1.AddToGroupVersion(scheme, SchemeBuilder.GroupVersion)
		return nil
	})
	SchemeBuilder.AddToScheme(runtimeScheme)
}
