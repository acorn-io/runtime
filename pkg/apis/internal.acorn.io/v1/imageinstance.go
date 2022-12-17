package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ContainerData struct {
	Image    string               `json:"image,omitempty"`
	Sidecars map[string]ImageData `json:"sidecars,omitempty"`
}

type ImageData struct {
	Image string `json:"image,omitempty"`
}

type ImagesData struct {
	Containers map[string]ContainerData `json:"containers,omitempty"`
	Jobs       map[string]ContainerData `json:"jobs,omitempty"`
	Images     map[string]ImageData     `json:"images,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Digest string   `json:"digest,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

func (in *ImageInstance) ShortID() string {
	if len(in.UID) > 11 {
		return string(in.UID[:12])
	}
	return string(in.UID)
}
