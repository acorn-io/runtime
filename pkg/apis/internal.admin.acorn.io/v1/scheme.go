package v1

import (
	internal_admin_acorn_io "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const Version = "v1"

var SchemeGroupVersion = schema.GroupVersion{
	Group:   internal_admin_acorn_io.Group,
	Version: Version,
}

func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ProjectVolumeClassInstance{},
		&ProjectVolumeClassInstanceList{},
		&ClusterVolumeClassInstance{},
		&ClusterVolumeClassInstanceList{},
		&ClusterComputeClassInstance{},
		&ClusterComputeClassInstanceList{},
		&ProjectComputeClassInstance{},
		&ProjectComputeClassInstanceList{})

	// Add common types
	scheme.AddKnownTypes(SchemeGroupVersion, &metav1.Status{})

	// Add the watch version that applies
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
