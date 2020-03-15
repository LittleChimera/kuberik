package v1alpha1

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// Screenplay describes how pipeline execution will look like
type Screenplay struct {
	Name                 string                         `json:"-" yaml:"-"`
	Scenes               []Scene                        `json:"scenes,omitempty"`
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
	Vars                 Vars                           `json:"vars,omitempty"`
}

// Var is a parametrizable variable for the screenplay shared between all jobs.
// +k8s:openapi-gen=true
type Var struct {
	Name      string     `json:"name"`
	Value     string     `json:"value,omitempty"`
	ValueFrom *VarSource `json:"valueFrom,omitempty"`
}

type Vars []Var

func (vars Vars) Get(name string) (string, error) {
	for _, v := range vars {
		if v.Name == name {
			return v.Value, nil
		}
	}
	return "", fmt.Errorf("Variable not found")
}

func (vars Vars) Set(name, value string) error {
	for i, v := range vars {
		if v.Name == name {
			vars[i].Value = value
			return nil
		}
	}
	return fmt.Errorf("Variable not declared")
}

// VarSource represents a source for the value of an Var.
type VarSource struct {
	// Selects a key of a ConfigMap.
	// +optional
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// Selects a key of a secret in the pod's namespace
	// +optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
	// Selects a path in input payload
	// +optional
	InputRef *InputFieldSelector `json:"inputRef,omitempty"`
}

// InputFieldSelector selects a path from input payload object.
type InputFieldSelector struct {
	GJSONPath string `json:"gjsonPath"`
}

// Scene finds a scene by name
func (s *Screenplay) Scene(name string) (*Scene, error) {
	for _, a := range s.Scenes {
		if a.Name == name {
			return &a, nil
		}
	}
	return &Scene{}, fmt.Errorf("Scene not found")
}

// Scene describes a collection of frames that need to be executed in parallel
type Scene struct {
	Name         string    `json:"name"`
	Frames       []Frame   `json:"frames"`
	Pass         Condition `json:"pass,omitempty"`
	When         Condition `json:"when,omitempty"`
	IgnoreErrors bool      `json:"ignore_errors,omitempty" yaml:"ignore_errors"`
}

// Condition describes a logical filter which controls execution of the pipeline
type Condition []map[string]string

// Evaluate returns the result of condition filter
func (c Condition) Evaluate(vars Vars) bool {
	var pass bool
	for _, conditions := range c {
		conditionPass := true
		for variable, v := range conditions {
			varValue, err := vars.Get(variable)

			if err != nil {
				conditionPass = conditionPass && false
				// TODO process error
				break
			}

			if varValue != v {
				conditionPass = conditionPass && false
				break
			}
		}
		pass = pass || conditionPass
	}
	return pass
}

// Frame describes either an action or story that needs to be executed
type Frame struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	IgnoreErrors bool   `json:"ignoreErrors,omitempty"`
	Copies       int    `json:"copies,omitempty"`

	Action Exec `json:"action,omitempty"`

	// OpenAPI doesn't generate this correctly because it's recursive
	// Needs to be manually validated in validating webhook
	// +kubebuilder:validation:Type=object
	Story Screenplay `json:"story,omitempty"`
}

// Exec Represents a running container
type Exec = batchv1.JobSpec

// Copy makes a copy of the frame
func (f *Frame) Copy() Frame {
	return Frame{
		ID:           f.ID,
		Name:         f.Name,
		Action:       *f.Action.DeepCopy(),
		IgnoreErrors: f.IgnoreErrors,
		Copies:       f.Copies,
	}
}
