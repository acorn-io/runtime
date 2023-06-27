package v1

import (
	"fmt"
	"net/url"

	api_acorn_io "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
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
		&ImageCopy{},
		&Info{},
		&InfoList{},
		&LogOptions{},
		&Volume{},
		&VolumeList{},
		&VolumeClass{},
		&VolumeClassList{},
		&Credential{},
		&CredentialList{},
		&CryptoKey{},
		&CryptoKeyList{},
		&ContainerReplica{},
		&ContainerReplicaList{},
		&ContainerReplicaExecOptions{},
		&ContainerReplicaPortForwardOptions{},
		&Secret{},
		&SecretList{},
		&Service{},
		&ServiceList{},
		&Project{},
		&ProjectList{},
		&AcornImageBuild{},
		&AcornImageBuildList{},
		&ComputeClass{},
		&ComputeClassList{},
		&Region{},
		&RegionList{},
		&ImageAllowRule{},
		&ImageAllowRuleList{},
		&Event{},
		&EventList{},
		&DevSession{},
		&DevSessionList{},
		&IgnoreCleanup{},
	)

	// Add common types
	scheme.AddKnownTypes(schemeGroupVersion, &metav1.Status{})

	if schemeGroupVersion == SchemeGroupVersion {
		// Add the watch version that applies
		metav1.AddToGroupVersion(scheme, schemeGroupVersion)

		if err := scheme.AddConversionFunc((*url.Values)(nil), (*ContainerReplicaPortForwardOptions)(nil), Convert_url_Values_To__ContainerReplicaPortForwardOptions); err != nil {
			return err
		}
		if err := scheme.AddConversionFunc((*url.Values)(nil), (*ContainerReplicaExecOptions)(nil), Convert_url_Values_To__ContainerReplicaExecOptions); err != nil {
			return err
		}
		if err := scheme.AddConversionFunc((*url.Values)(nil), (*LogOptions)(nil), Convert_url_Values_To__LogOptions); err != nil {
			return err
		}

		gvk := schemeGroupVersion.WithKind("Event")
		flcf := func(label, value string) (string, string, error) {
			switch label {
			case "prefix", "since", "until", "details", "metadata.name", "metadata.namespace":
				return label, value, nil
			}
			return "", "", fmt.Errorf("unsupported field selection [%s]", label)
		}
		if err := scheme.AddFieldLabelConversionFunc(gvk, flcf); err != nil {
			return err
		}
	}

	return nil
}
