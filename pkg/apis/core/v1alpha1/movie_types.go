package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MovieSpec defines the desired state of Movie
// +k8s:openapi-gen=true
type MovieSpec struct {
	Template PlayTemplate `json:"template"`
	// +optional
	FailedJobsHistoryLimit int `json:"failedJobsHistoryLimit"`
	// +optional
	SuccessfulJobsHistoryLimit int `json:"successfulJobsHistoryLimit"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// PlayTemplate defines a template of Play to be created from a Movie
type PlayTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PlaySpec `json:"spec,omitempty"`
}

// MovieStatus defines the observed state of Movie
// +k8s:openapi-gen=true
type MovieStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Movie is the Schema for the movies API
// +k8s:openapi-gen=true
// +genclient
// +kubebuilder:subresource:status
type Movie struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MovieSpec   `json:"spec,omitempty"`
	Status MovieStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MovieList contains a list of Movie
type MovieList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Movie `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Movie{}, &MovieList{})
}
