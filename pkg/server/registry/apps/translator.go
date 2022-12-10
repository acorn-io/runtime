package apps

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	mtypes "github.com/acorn-io/mink/pkg/types"
)

type Translator struct {
}

func (s *Translator) FromPublic(obj mtypes.Object) mtypes.Object {
	app := obj.(*apiv1.App)
	return &v1.AppInstance{
		ObjectMeta: app.ObjectMeta,
		Spec:       app.Spec,
		Status:     app.Status,
	}
}

func (s *Translator) ToPublic(obj mtypes.Object) mtypes.Object {
	app := obj.(*v1.AppInstance)
	return &apiv1.App{
		ObjectMeta: app.ObjectMeta,
		Spec:       app.Spec,
		Status:     app.Status,
	}
}
