package images

import (
	mtypes "github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

type Translator struct {
}

func (s *Translator) FromPublic(obj mtypes.Object) mtypes.Object {
	image := obj.(*apiv1.Image)
	return (*v1.ImageInstance)(image)
}

func (s *Translator) ToPublic(obj mtypes.Object) mtypes.Object {
	image := obj.(*v1.ImageInstance)
	return (*apiv1.Image)(image)
}
