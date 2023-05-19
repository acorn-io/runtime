package events

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	mtypes "github.com/acorn-io/mink/pkg/types"
)

type translator struct{}

func (s *translator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*v1.EventInstance)(obj.(*apiv1.Event))
}

func (s *translator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*apiv1.Event)(obj.(*v1.EventInstance))
}
