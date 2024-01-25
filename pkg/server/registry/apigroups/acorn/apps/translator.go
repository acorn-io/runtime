package apps

import (
	mtypes "github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

type translator struct{}

func (*translator) FromPublic(obj mtypes.Object) mtypes.Object {
	return apiv1.AppToAppInstance(obj.(*apiv1.App))
}

func (*translator) ToPublic(obj mtypes.Object) mtypes.Object {
	return apiv1.AppInstanceToApp(obj.(*v1.AppInstance))
}
