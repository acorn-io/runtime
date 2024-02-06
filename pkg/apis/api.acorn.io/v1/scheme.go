package v1

import (
	"fmt"
	"net/url"

	apiacornio "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const Version = "v1"

var SchemeGroupVersion = schema.GroupVersion{
	Group:   apiacornio.Group,
	Version: Version,
}

func AddToScheme(scheme *runtime.Scheme) error {
	return AddToSchemeWithGV(scheme, SchemeGroupVersion)
}

func AddToSchemeWithGV(scheme *runtime.Scheme, schemeGroupVersion schema.GroupVersion) error {
	scheme.AddKnownTypes(schemeGroupVersion,
		&App{},
		&AppList{},
		&AppInfo{},
		&Builder{},
		&BuilderPortOptions{},
		&BuilderList{},
		&ConfirmUpgrade{},
		&AppPullImage{},
		&IconOptions{},
		&Image{},
		&ImageList{},
		&ImageDetails{},
		&ImageTag{},
		&ImagePush{},
		&ImagePull{},
		&ImageSignature{},
		&Info{},
		&InfoList{},
		&LogOptions{},
		&Volume{},
		&VolumeList{},
		&VolumeClass{},
		&VolumeClassList{},
		&Credential{},
		&CredentialList{},
		&ContainerReplica{},
		&ContainerReplicaList{},
		&ContainerReplicaExecOptions{},
		&ContainerReplicaPortForwardOptions{},
		&Job{},
		&JobRestart{},
		&JobList{},
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

		if err := scheme.AddConversionFunc((*url.Values)(nil), (*ContainerReplicaPortForwardOptions)(nil), ConvertURLValuesToContainerReplicaPortForwardOptions); err != nil {
			return err
		}
		if err := scheme.AddConversionFunc((*url.Values)(nil), (*ContainerReplicaExecOptions)(nil), ConvertURLValuesToContainerReplicaExecOptions); err != nil {
			return err
		}
		if err := scheme.AddConversionFunc((*url.Values)(nil), (*LogOptions)(nil), ConvertURLValuesToLogOptions); err != nil {
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
