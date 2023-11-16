package v1

import corev1 "k8s.io/api/core/v1"

const (
	SecretTypePrefix                             = "secrets.acorn.io/"
	SecretTypeOpaque           corev1.SecretType = "secrets.acorn.io/opaque"
	SecretTypeGenerated        corev1.SecretType = "secrets.acorn.io/generated"
	SecretTypeTemplate         corev1.SecretType = "secrets.acorn.io/template"
	SecretTypeBasic            corev1.SecretType = "secrets.acorn.io/basic"
	SecretTypeToken            corev1.SecretType = "secrets.acorn.io/token"
	SecretTypeCredentialPrefix                   = "credential."
)

var (
	SecretTypes = map[corev1.SecretType]bool{
		SecretTypeOpaque:    true,
		SecretTypeGenerated: true,
		SecretTypeTemplate:  true,
		SecretTypeBasic:     true,
		SecretTypeToken:     true,
	}
)
