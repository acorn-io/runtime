package regions

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	mtypes "github.com/acorn-io/mink/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Translator struct{}

func (s *Translator) FromPublic(obj mtypes.Object) mtypes.Object {
	region := (*v1.RegionInstance)(obj.(*apiv1.Region))
	region.TypeMeta = metav1.TypeMeta{}
	return region
}

func (s *Translator) ToPublic(obj mtypes.Object) mtypes.Object {
	region := (*apiv1.Region)(obj.(*v1.RegionInstance))
	region.TypeMeta = metav1.TypeMeta{}
	return region
}
