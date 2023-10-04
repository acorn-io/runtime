// +k8s:deepcopy-gen=package

package v1

import (
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoleAuthorizations struct {
	Scopes   []string  `json:"scopes,omitempty"`
	RoleRefs []RoleRef `json:"roleRefs,omitempty"`
}

type RoleRef struct {
	Name string `json:"name,omitempty"`
	Kind string `json:"kind,omitempty"` // ClusterRole (or Role - if not cluster-scoped)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterImageRoleAuthorizationInstance ImageRoleAuthorizationInstance

func (in *ClusterImageRoleAuthorizationInstance) NamespaceScoped() bool {
	return false
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterImageRoleAuthorizationInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterImageRoleAuthorizationInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageRoleAuthorizationInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ImageRoleAuthorizationInstanceSpec   `json:"spec,omitempty"`
	Status ImageRoleAuthorizationInstanceStatus `json:"status,omitempty"`
}

type ImageRoleAuthorizationInstanceSpec struct {
	ImageSelector internalv1.ImageSelector `json:"imageSelector,omitempty"`
	Roles         RoleAuthorizations       `json:"roles,omitempty"`
}

type ImageRoleAuthorizationInstanceStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageRoleAuthorizationInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageRoleAuthorizationInstance `json:"items"`
}
