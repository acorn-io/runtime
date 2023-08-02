package v1

import (
	"time"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Event internalv1.EventInstance

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Event `json:"items"`
}

// Alias helper types so that clients don't need to import the internal package when creating events.

// +k8s:deepcopy-gen=false

type EventResource = internalv1.EventResource

// +k8s:deepcopy-gen=false

type MicroTime = internalv1.MicroTime

func NowMicro() MicroTime {
	return internalv1.NowMicro()
}

func NewMicroTime(t time.Time) MicroTime {
	return internalv1.NewMicroTime(t)
}

const (
	EventSeverityInfo  = internalv1.EventSeverityInfo
	EventSeverityError = internalv1.EventSeverityError
)

func Mapify(v any) (internalv1.GenericMap, error) {
	return internalv1.Mapify(v)
}
