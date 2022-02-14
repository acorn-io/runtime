package appdefinition

import (
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/rancher/wrangler/pkg/name"
)

func AssignNamespace(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	if appInstance.Status.Namespace != "" {
		return nil
	}

	appInstance.Status.Namespace = name.SafeConcatName(appInstance.Name, string(appInstance.UID)[:8])
	resp.Objects(appInstance)
	return nil
}

func RequireNamespace(h router.HandlerFunc) router.HandlerFunc {
	return func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		if appInstance.Status.Namespace == "" {
			return nil
		}
		return h(req, resp)
	}
}
