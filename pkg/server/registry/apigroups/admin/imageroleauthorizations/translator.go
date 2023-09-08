package imageroleauthorizations

import (
	mtypes "github.com/acorn-io/mink/pkg/types"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
)

type Translator struct{}

func (s *Translator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*internaladminv1.ImageRoleAuthorizationInstance)(obj.(*adminv1.ImageRoleAuthorization))
}

func (s *Translator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.ImageRoleAuthorization)(obj.(*internaladminv1.ImageRoleAuthorizationInstance))
}

type ClusterTranslator struct{}

func (s *ClusterTranslator) FromPublic(obj mtypes.Object) mtypes.Object {
	return (*internaladminv1.ClusterImageRoleAuthorizationInstance)(obj.(*adminv1.ClusterImageRoleAuthorization))
}
func (s *ClusterTranslator) ToPublic(obj mtypes.Object) mtypes.Object {
	return (*adminv1.ClusterImageRoleAuthorization)(obj.(*internaladminv1.ClusterImageRoleAuthorizationInstance))
}
