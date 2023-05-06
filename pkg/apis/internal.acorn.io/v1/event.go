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
	// +required
	Type string `json:"type,omitempty"`

	// Actor is the ID of the entity that generated the Event.
	// This can be the name of a particular user or controller.
	// +required
	Actor string `json:"actor"`

	// Subject is the object the Event is regarding.
	// +optional
	Subject *EventSubject `json:"subject,omitempty"`

	// Details is a human-readable description of the Event.
	// +optional
	Details string `json:"details,omitempty"`

	// Time represents the time the Event was first observed.
	// +optional
	Time *metav1.MicroTime `json:"time,omitempty"`
}

// EventSubject describes an object related to an Event.
// It can contain one of:
// - a reference to the object
// - the object in its entirety
//
// Note: corev1.ObjectReference was explicitly avoided because its use in new schemas is discouraged.
// See https://github.com/kubernetes/api/blob/cdff1d4efea5d7ddc52c4085f82748c5f3e5cc8e/core/v1/types.go#L5919
// for more details.
type EventSubject struct {
	// Type identifies the type of the EventSubject.
	// +unionDiscriminator
	Type EventSubjectType `json:"type"`

	// Reference is a reference to the event EventSubject's object.
	// +optional
	Reference *EventSubjectReference `json:"reference,omitempty"`

	// Object is a reference to the event EventSubject's object.
	// +optional
	Object *EventSubjectObject `json:"object,omitempty"`
}

// EventSubjectType identifies a type of EventSubject.
type EventSubjectType string

const (
	// EventSubjectTypeReference identifies a reference to an object.
	EventSubjectTypeReference EventSubjectType = "Reference"

	EventSubjectTypeObject EventSubjectType = "Object"
)

type EventSubjectReference struct {
	// TODO(njhale): Implement me!
}

type EventSubjectObject struct {
	// TODO(njhale): Implement me!
}
