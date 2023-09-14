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

type ImageSelector struct {
	NamePatterns []string         `json:"namePatterns,omitempty"`
	Signatures   []SignatureRules `json:"signatures,omitempty"`
}
