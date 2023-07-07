package projects

import (
	mtypes "github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

type translator struct{}

func (t *translator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*apiv1.Project)(obj.(*v1.ProjectInstance))
}

func (t *translator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*v1.ProjectInstance)(obj.(*apiv1.Project))
}
