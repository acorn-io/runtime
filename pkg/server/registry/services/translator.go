package services

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/mink/pkg/types"
)

type Translator struct {
}

func (t *Translator) FromPublic(obj types.Object) types.Object {
	result := (*v1.ServiceInstance)(obj.(*apiv1.Service))
	return result
}

func (t *Translator) ToPublic(obj types.Object) types.Object {
	result := (*apiv1.Service)(obj.(*v1.ServiceInstance))
	return result
}
