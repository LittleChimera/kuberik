package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PlaySpec defines the desired state of Play
// +k8s:openapi-gen=true
type PlaySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Screenplays          []Screenplay                   `json:"screenplays"`
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
	Vars                 Vars                           `json:"vars,omitempty"`
}

// PlayStatus defines the observed state of Play
// +k8s:openapi-gen=true
type PlayStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Frames             map[string]int    `json:"frames,omitempty"`
	Phase              PlayPhaseType     `json:"phase,omitempty"`
	Runner             string            `json:"runner,omitempty"`
	ProvisionedVolumes map[string]string `json:"provisionedVolumes,omitempty"`
	VarsConfigMap      string            `json:"varsConfigMap,omitempty"`
}

// PlayPhaseType defines the phase of a Play
type PlayPhaseType string

// These are valid phases of a play.
const (
	// PlayComplete means the play has completed its execution.
	PlayComplete PlayPhaseType = "Complete"
	// PlayFailed means the play has failed its execution.
	PlayFailed PlayPhaseType = "Failed"
	// PlayRunning means the play is executing.
	PlayRunning PlayPhaseType = "Running"
	// PlayRunning means the play has been created.
	PlayCreated PlayPhaseType = "Created"
	// PlayError means the play ended because of an error.
	PlayError PlayPhaseType = "Error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Play is the Schema for the plays API
// +k8s:openapi-gen=true
// +genclient
// +kubebuilder:subresource:status
type Play struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlaySpec   `json:"spec,omitempty"`
	Status PlayStatus `json:"status,omitempty"`
}

// Frame gets a frame with requested identifier
func (p *Play) Frame(frameID string) *Frame {
	for spi, screenplay := range p.Spec.Screenplays {
		for sci, scene := range screenplay.Scenes {
			for fi, frame := range scene.Frames {
				if frame.ID == frameID {
					return &p.Spec.Screenplays[spi].Scenes[sci].Frames[fi]
				}
			}
		}
	}
	return nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlayList contains a list of Play
type PlayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Play `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Play{}, &PlayList{})
}
