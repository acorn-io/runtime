package v1

import (
	acorn_io "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const Version = "v1"

var SchemeGroupVersion = schema.GroupVersion{
	Group:   acorn_io.Group,
	Version: Version,
}

func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&AcornImageBuildInstance{},
		&AcornImageBuildInstanceList{},
		&BuilderInstance{},
		&BuilderInstanceList{},
		&AppInstance{},
		&AppInstanceList{},
		&ServiceInstance{},
		&ServiceInstanceList{},
		&ImageInstance{},
		&ImageInstanceList{},
		&ImageAllowRuleInstance{},
		&ImageAllowRuleInstanceList{},
		&EventInstance{},
		&EventInstanceList{},
		&DevSessionInstance{},
		&DevSessionInstanceList{},
		&ProjectInstance{},
		&ProjectInstanceList{},
		&ImageMetadataCache{},
		&ImageMetadataCacheList{},
	)

	// Add common types
	scheme.AddKnownTypes(SchemeGroupVersion, &metav1.Status{})

	// Add the watch version that applies
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
