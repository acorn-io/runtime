package events

import (
	mtypes "github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

type translator struct{}

func (s *translator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*v1.EventInstance)(obj.(*apiv1.Event))
}

func (s *translator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*apiv1.Event)(obj.(*v1.EventInstance))
}
