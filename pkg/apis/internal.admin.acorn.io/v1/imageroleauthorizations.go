// +k8s:deepcopy-gen=package

package v1

import (
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoleRef struct {
	RoleName string `json:"role,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterImageRoleAuthorizationInstance ImageRoleAuthorizationInstance

func (in *ClusterImageRoleAuthorizationInstance) NamespaceScoped() bool {
	return false
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageRoleAuthorizationInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Images     []string                            `json:"images,omitempty"` // list of patterns to match against image names
	Signatures internalv1.ImageAllowRuleSignatures `json:"signatures,omitempty"`
	RoleRefs   []RoleRef                           `json:"roleRefs,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageRoleAuthorizationInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageRoleAuthorizationInstance `json:"items"`
}
