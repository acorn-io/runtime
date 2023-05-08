package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EventInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EventInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Type is a short, machine-readable string that describes the kind of Event that took place.
	Type string `json:"type"`

	// Actor is the ID of the entity that generated the Event.
	// This can be the name of a particular user or controller.
	Actor string `json:"actor"`

	// Subject identifies the object the Event is regarding.
	Subject EventSubject `json:"subject"`

	// Context provides additional information about the cluster at the time the Event occurred.
	//
	// It's typically used to embed the subject resource, in its entirety, at the time the Event occurred,
	// but can be used to hold any data related to the event.
	//
	// +optional
	Context GenericMap `json:"context,omitempty"`

	// Description is a human-readable description of the Event.
	// +optional
	Description *string `json:"description,omitempty"`

	// Observed represents the time the Event was first observed.
	Observed metav1.MicroTime `json:"observed"`
}

// EventSubject identifies an object related to an Event.
//
// The referenced object may or may not exist.
//
// Note: corev1.ObjectReference was explicitly avoided because its use in new schemas is discouraged.
// See https://github.com/kubernetes/api/blob/cdff1d4efea5d7ddc52c4085f82748c5f3e5cc8e/core/v1/types.go#L5919
// for more details.
type EventSubject struct {
	// Kind is the kind of the subject.
	Kind string `json:"kind"`

	// Name is the name of the subject.
	Name string `json:"name"`
}
