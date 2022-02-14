package v1

import (
	"github.com/ibuildthecloud/herd/pkg/apis/herd-project.io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const Version = "v1"

var SchemeGroupVersion = schema.GroupVersion{
	Group:   herd_project_io.Group,
	Version: Version,
}

func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&AppInstance{},
		&AppInstanceList{})

	// Add common types
	scheme.AddKnownTypes(SchemeGroupVersion, &metav1.Status{})

	// Add the watch version that applies
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
