package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Condition struct {
	Type               string                 `json:"type,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
	Status             metav1.ConditionStatus `json:"status,omitempty"`
	ObservedGeneration int64                  `json:"observedGeneration,omitempty"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`

	Success       bool `json:"success,omitempty"`
	Error         bool `json:"error,omitempty"`
	Transitioning bool `json:"transitioning,omitempty"`
}

func (in Condition) Init(name string, generation int64) Condition {
	return Condition{
		Type: name,
	}.Set(in, generation)
}

func (in Condition) ToReason() string {
	if in.Transitioning {
		return "InProgress"
	}
	if in.Error {
		return "Error"
	}
	if in.Success {
		return "Success"
	}
	return "Success"
}

func (in Condition) ToStatus() metav1.ConditionStatus {
	if in.Transitioning {
		return metav1.ConditionUnknown
	}
	if in.Error {
		return metav1.ConditionFalse
	}
	if in.Success {
		return metav1.ConditionTrue
	}
	return metav1.ConditionTrue
}

func (in Condition) Set(cond Condition, generation int64) Condition {
	if in.Type == "" {
		panic("type must be set on condition")
	}
	if in.Success == cond.Success &&
		in.Error == cond.Error &&
		in.Transitioning == cond.Transitioning &&
		in.Message == cond.Message {
		return in
	}

	return Condition{
		Type:               in.Type,
		Message:            cond.Message,
		Status:             cond.ToStatus(),
		Reason:             cond.ToReason(),
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Success:            cond.Success,
		Error:              cond.Error,
		Transitioning:      cond.Transitioning,
	}
}
