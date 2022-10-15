package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AcornBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   AcornBuildSpec   `json:"spec,omitempty"`
	Status AcornBuildStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AcornBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AcornBuild `json:"items"`
}

type AcornImageBuild struct {
	GitRepoURL     string            `json:"gitRepoURL,omitempty"`
	Context        string            `json:"context,omitempty"`
	AcornBuildArgs map[string]string `json:"buildArgs,omitempty"`
}

type GitRepoWatch struct {
	Revision string `json:"revision,omitempty"`
	Branch   string `json:"branch,omitempty"`
	PR       bool   `json:"pr,omitempty"`
}

type AcornBuildPush struct {
	Registry  string `json:"registry,omitempty"`
	ImageName string `json:"imageName,omitempty"`
}

type AcornBuildSpec struct {
	Build AcornImageBuild `json:"build,omitempty"`
	Watch GitRepoWatch    `json:"watch,omitempty"`
	Push  AcornBuildPush  `json:"push,omitempty"`
}

type AcornBuildStatus struct {
	ClusterName string `json:"clusterName,omitempty"`
}
