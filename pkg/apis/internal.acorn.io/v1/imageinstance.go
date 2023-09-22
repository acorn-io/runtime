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
	Acorns     map[string]ImageData     `json:"acorns,omitempty"`
	Builds     []BuildRecord            `json:"builds,omitempty"`
}

type BuildRecord struct {
	AcornBuild     *AcornBuilderSpec          `json:"acornBuild,omitempty"`
	AcornAppImage  *AppImage                  `json:"acornAppImage,omitempty"`
	ContainerBuild *ContainerImageBuilderSpec `json:"containerBuild,omitempty"`
	ImageBuild     *ImageBuilderSpec          `json:"imageBuild,omitempty"`
	ImageKey       string                     `json:"imageKey,omitempty"`
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

	// Remote indicates that this image has not been locally cached to the internal registry
	// meaning that it may not exist at the location recorded in the Repo field if the user
	// has deleted the image after the fact
	ZZ_Remote bool     `json:"remote,omitempty"`
	Repo      string   `json:"repo,omitempty"`
	Digest    string   `json:"digest,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

func (in *ImageInstance) ShortID() string {
	if len(in.UID) > 11 {
		return string(in.UID[:12])
	}
	return string(in.UID)
}
