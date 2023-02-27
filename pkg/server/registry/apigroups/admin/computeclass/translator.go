package computeclass

import (
	adminv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	adminapiv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"

	mtypes "github.com/acorn-io/mink/pkg/types"
)

type ClusterTranslator struct{}

func (s *ClusterTranslator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*adminapiv1.ClusterComputeClassInstance)(obj.(*adminv1.ClusterComputeClass))
}
func (s *ClusterTranslator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.ClusterComputeClass)(obj.(*adminapiv1.ClusterComputeClassInstance))
}

type ProjectTranslator struct{}

func (s *ProjectTranslator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*adminapiv1.ProjectComputeClassInstance)(obj.(*adminv1.ProjectComputeClass))
}
func (s *ProjectTranslator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.ProjectComputeClass)(obj.(*adminapiv1.ProjectComputeClassInstance))
}
