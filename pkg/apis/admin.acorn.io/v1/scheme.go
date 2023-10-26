package v1

import (
	admin_acorn_io "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const Version = "v1"

var SchemeGroupVersion = schema.GroupVersion{
	Group:   admin_acorn_io.Group,
	Version: Version,
}

func AddToScheme(scheme *runtime.Scheme) error {
	return AddToSchemeWithGV(scheme, SchemeGroupVersion)
}

func AddToSchemeWithGV(scheme *runtime.Scheme, schemeGroupVersion schema.GroupVersion) error {
	scheme.AddKnownTypes(schemeGroupVersion,
		&ProjectVolumeClass{},
		&ProjectVolumeClassList{},
		&ClusterVolumeClass{},
		&ClusterVolumeClassList{},
		&ClusterComputeClass{},
		&ClusterComputeClassList{},
		&ProjectComputeClass{},
		&ProjectComputeClassList{},
		&ImageRoleAuthorization{},
		&ImageRoleAuthorizationList{},
		&ClusterImageRoleAuthorization{},
		&ClusterImageRoleAuthorizationList{},
		&QuotaRequest{},
		&QuotaRequestList{},
	)

	// Add common types
	scheme.AddKnownTypes(schemeGroupVersion, &metav1.Status{})

	if schemeGroupVersion == SchemeGroupVersion {
		// Add the watch version that applies
		metav1.AddToGroupVersion(scheme, schemeGroupVersion)
	}
	return nil
}
