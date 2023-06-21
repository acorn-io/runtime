package volumeclass

import (
	mtypes "github.com/acorn-io/mink/pkg/types"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	admininternalv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
)

type ClusterTranslator struct{}

func (s *ClusterTranslator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*admininternalv1.ClusterVolumeClassInstance)(obj.(*adminv1.ClusterVolumeClass))
}
func (s *ClusterTranslator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.ClusterVolumeClass)(obj.(*admininternalv1.ClusterVolumeClassInstance))
}

type ProjectTranslator struct{}

func (s *ProjectTranslator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*admininternalv1.ProjectVolumeClassInstance)(obj.(*adminv1.ProjectVolumeClass))
}
func (s *ProjectTranslator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.ProjectVolumeClass)(obj.(*admininternalv1.ProjectVolumeClassInstance))
}
