package keys

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	mtypes "github.com/acorn-io/mink/pkg/types"
)

type Translator struct{}

func (s *Translator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*internalv1.PublicKeyInstance)(obj.(*apiv1.PublicKey))
}

func (s *Translator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*apiv1.PublicKey)(obj.(*internalv1.PublicKeyInstance))
}
