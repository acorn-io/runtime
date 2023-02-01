package workloadclass

import (
	adminv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	adminapiv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"

	mtypes "github.com/acorn-io/mink/pkg/types"
)

type ClusterTranslator struct{}

func (s *ClusterTranslator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*adminapiv1.ClusterWorkloadClassInstance)(obj.(*adminv1.ClusterWorkloadClass))
}
func (s *ClusterTranslator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.ClusterWorkloadClass)(obj.(*adminapiv1.ClusterWorkloadClassInstance))
}

type ProjectTranslator struct{}

func (s *ProjectTranslator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*adminapiv1.ProjectWorkloadClassInstance)(obj.(*adminv1.ProjectWorkloadClass))
}
func (s *ProjectTranslator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.ProjectWorkloadClass)(obj.(*adminapiv1.ProjectWorkloadClassInstance))
}
