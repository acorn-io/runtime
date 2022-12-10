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
	return AddToSchemeWithGV(scheme, SchemeGroupVersion)
}

func AddToSchemeWithGV(scheme *runtime.Scheme, schemeGroupVersion schema.GroupVersion) error {
	scheme.AddKnownTypes(schemeGroupVersion,
		&App{},
		&AppList{},
		&Builder{},
		&BuilderPortOptions{},
		&BuilderList{},
		&ConfirmUpgrade{},
		&AppPullImage{},
		&Image{},
		&ImageList{},
		&ImageDetails{},
		&ImageTag{},
		&ImagePush{},
		&ImagePull{},
		&Info{},
		&InfoList{},
		&LogOptions{},
		&Volume{},
		&VolumeList{},
		&Credential{},
		&CredentialList{},
		&ContainerReplica{},
		&ContainerReplicaList{},
		&ContainerReplicaExecOptions{},
		&Secret{},
		&SecretList{},
		&Project{},
		&ProjectList{},
	)

	// Add common types
	scheme.AddKnownTypes(schemeGroupVersion, &metav1.Status{})

	if schemeGroupVersion == SchemeGroupVersion {
		// Add the watch version that applies
		metav1.AddToGroupVersion(scheme, schemeGroupVersion)

		if err := scheme.AddConversionFunc((*url.Values)(nil), (*ContainerReplicaExecOptions)(nil), Convert_url_Values_To__ContainerReplicaExecOptions); err != nil {
			return err
		}
		return scheme.AddConversionFunc((*url.Values)(nil), (*LogOptions)(nil), Convert_url_Values_To__LogOptions)
	}

	return nil
}
