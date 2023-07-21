package v1

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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

	// Severity indicates the severity of the event.
	// +optional
	Severity EventSeverity `json:"severity,omitempty"`

	// Actor is the ID of the entity that generated the Event.
	// This can be the name of a particular user or controller.
	// +optional
	Actor string `json:"actor,omitempty"`

	// AppName is the name of the app that the event is related to, if any.
	// +optional
	AppName string `json:"appName,omitempty"`

	// Resource identifies the object the Event is regarding.
	// +optional
	Resource *EventResource `json:"resource,omitempty"`

	// Description is a human-readable description of the Event.
	// +optional
	Description string `json:"description,omitempty"`

	// Observed represents the time the Event was first observed.
	// +optional
	Observed MicroTime `json:"observed,omitempty" wrangler:"type=string"`

	// Details provides additional information about the cluster at the time the Event occurred.
	//
	// It's typically used to embed the subject resource, in its entirety, at the time the Event occurred,
	// but can be used to hold any data related to the event.
	//
	// +optional
	Details GenericMap `json:"details,omitempty"`
}

// GetObserved returns the time that the Event was first observed.
func (e EventInstance) GetObserved() MicroTime {
	return e.Observed
}

// MicroTime represents a time with microsecond level precision.
//
// It extends metav1.MicroTime to allow unmarshaling from RFC3339.
type MicroTime metav1.MicroTime

func NewMicroTime(t time.Time) MicroTime {
	return MicroTime(metav1.NewMicroTime(t))
}

func NowMicro() MicroTime {
	return NewMicroTime(time.Now())
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ MicroTime) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ MicroTime) OpenAPISchemaFormat() string { return "date-time" }

// DeepCopyInto returns a deep-copy of the MicroTime value.  The underlying time.Time
// type is effectively immutable in the time API, so it is safe to
// copy-by-assign, despite the presence of (unexported) Pointer fields.
func (t *MicroTime) DeepCopyInto(out *MicroTime) {
	*out = *t
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (t *MicroTime) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		t.Time = time.Time{}
		return nil
	}

	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	pt, err := time.Parse(metav1.RFC3339Micro, str)
	if err != nil {
		var sErr error
		if pt, sErr = time.Parse(time.RFC3339, str); sErr != nil {
			return errors.Join(err, sErr)
		}
	}

	t.Time = pt.Local()
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (t MicroTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		// Encode unset/nil objects as JSON's "null".
		return []byte("null"), nil
	}

	return json.Marshal(t.UTC().Format(metav1.RFC3339Micro))
}

const (
	// EventSeverityInfo indicates an event describes a system operating "as expected".
	EventSeverityInfo EventSeverity = "info"

	// EventSeverityError indicates an event describes an error.
	EventSeverityError EventSeverity = "error"
)

// EventSeverity indicates the severity of an event.
type EventSeverity string

// EventResource identifies a resource related to an Event.
//
// The referenced resource may or may not exist.
//
// Note: corev1.ObjectReference was explicitly avoided because its use in new schemas is discouraged.
// See https://github.com/kubernetes/api/blob/cdff1d4efea5d7ddc52c4085f82748c5f3e5cc8e/core/v1/types.go#L5919
// for more details.
type EventResource struct {
	// Kind is the resource kind.
	Kind string `json:"kind"`

	// Name is the name of the resource.
	Name string `json:"name"`

	// UID uniquely identifies the resource.
	UID types.UID `json:"uuid"`
}

func (e EventResource) String() string {
	components := make([]string, 0, 2)
	if e.Kind != "" {
		components = append(components, e.Kind)
	}
	if e.Name != "" {
		components = append(components, e.Name)
	}

	return strings.Join(components, "/")
}
