// +k8s:deepcopy-gen=package

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SignedBy struct {
	AnyOf []string `json:"anyOf,omitempty"`
	AllOf []string `json:"allOf,omitempty"`
}

type SignatureAnnotations struct {
	Match       map[string]string                 `json:"match,omitempty"`
	Expressions []metav1.LabelSelectorRequirement `json:"expressions,omitempty"`
}

type SignatureRules struct {
	SignedBy    SignedBy             `json:"signedBy,omitempty"`
	Annotations SignatureAnnotations `json:"annotations,omitempty"`
}

type ImageAllowRuleSignatures struct {
	Rules []SignatureRules `json:"rules,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageAllowRuleInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Images     []string                 `json:"images,omitempty"` // list of patterns to match against image names
	Signatures ImageAllowRuleSignatures `json:"signatures,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageAllowRuleInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageAllowRuleInstance `json:"items"`
}
