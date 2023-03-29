// +k8s:deepcopy-gen=package

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type SignedBy struct {
	AnyOf []string `json:"anyOf,omitempty"`
	AllOf []string `json:"allOf,omitempty"`
}

type SignatureAnnotations struct {
	Match       map[string]string                 `json:"match,omitempty"`
	Expressions []metav1.LabelSelectorRequirement `json:"expressions,omitempty"`
}

func (r *SignatureAnnotations) AsSelector() (labels.Selector, error) {
	labelselector := &metav1.LabelSelector{
		MatchLabels:      r.Match,
		MatchExpressions: r.Expressions,
	}

	return metav1.LabelSelectorAsSelector(labelselector)
}

type SignatureRules struct {
	SignedBy    SignedBy             `json:"signedBy,omitempty"`
	Annotations SignatureAnnotations `json:"annotations,omitempty"`
}

type ImageAllowRuleSignatures struct {
	Rules []SignatureRules `json:"rules,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageAllowRulesInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Signatures ImageAllowRuleSignatures `json:"signatures,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageAllowRulesInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageAllowRulesInstance `json:"items"`
}
