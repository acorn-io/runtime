package v1

import (
	"net/url"

	api_acorn_io "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const Version = "v1"

var SchemeGroupVersion = schema.GroupVersion{
	Group:   api_acorn_io.Group,
	Version: Version,
}

func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&App{},
		&AppList{},
		&Builder{},
		&BuilderPortOptions{},
		&BuilderList{},
		&Image{},
		&ImageList{},
		&ImageDetails{},
		&ImageTag{},
		&ImagePush{},
		&ImagePull{},
		&Info{},
		&InfoList{},
		&Volume{},
		&VolumeList{},
		&Credential{},
		&CredentialList{},
		&ContainerReplica{},
		&ContainerReplicaList{},
		&ContainerReplicaExecOptions{},
		&Secret{},
		&SecretList{},
	)

	// Add common types
	scheme.AddKnownTypes(SchemeGroupVersion, &metav1.Status{})

	// Add the watch version that applies
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return scheme.AddConversionFunc((*url.Values)(nil), (*ContainerReplicaExecOptions)(nil), Convert_url_Values_To__ContainerReplicaExecOptions)
}
