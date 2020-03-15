package operator

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	extensionsGroup = "extensions.kuberik.io"
)

type Resource struct {
	schema.GroupVersionKind
}

func (r Resource) listKind() string {
	return fmt.Sprintf("%sList", r.Kind)
}

func (r Resource) crdName() string {
	return fmt.Sprintf("%s.%s", r.plural(), extensionsGroup)
}

func (r Resource) singular() string {
	return strings.ToLower(r.Kind)
}

func (r Resource) plural() string {
	return fmt.Sprintf("%ss", r.singular())
}

func newResourceFromGVK(gvk schema.GroupVersionKind) Resource {
	gvk.Group = extensionsGroup
	return Resource{
		GroupVersionKind: gvk,
	}
}
