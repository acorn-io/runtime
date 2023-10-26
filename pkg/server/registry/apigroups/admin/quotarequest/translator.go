package quotarequest

import (
	mtypes "github.com/acorn-io/mink/pkg/types"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	admininternalv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
)

type Translator struct{}

func (s *Translator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*admininternalv1.QuotaRequestInstance)(obj.(*adminv1.QuotaRequest))
}
func (s *Translator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.QuotaRequest)(obj.(*admininternalv1.QuotaRequestInstance))
}
