package appstatus

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

func CheckStatus(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		conditionsToCheck := []string{
			v1.AppInstanceConditionDefaults,
			v1.AppInstanceConditionScheduling,
			v1.AppInstanceConditionQuota,
		}

		for _, cond := range conditionsToCheck {
			if !appInstance.Status.Condition(cond).Success {
				resp.DisablePrune()
				return nil
			}
		}

		return h.Handle(req, resp)
	})
}
