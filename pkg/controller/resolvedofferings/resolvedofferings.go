package resolvedofferings

import (
	"github.com/acorn-io/baaah/pkg/router"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/condition"
	"github.com/acorn-io/runtime/pkg/config"
)

// Calculate is a handler that sets the resolved offerings for an AppInstance to its status.
func Calculate(req router.Request, resp router.Response) (err error) {
	appInstance := req.Object.(*internalv1.AppInstance)
	status := condition.Setter(appInstance, resp, internalv1.AppInstanceConditionResolvedOfferings)

	defer func() {
		if err == nil {
			status.Success()
		} else {
			status.Error(err)
			// Disable prune because we are short-circuiting the handlers and don't want objects getting cleaned up accidentally.
			resp.DisablePrune()
			// Don't return the error so the other conditions don't get the same information.
			err = nil
		}
	}()

	// resolveVolumeClasses is idempotent and will only set volume class info if it is not already present.
	if err = resolveVolumeClasses(req.Ctx, req.Client, appInstance); err != nil {
		return err
	}

	// Calculate the resolved offerings for the AppInstance if its generation changed, or if at least one of
	// the containers has not been resolved yet. The generation check prevents the app from redeploying
	// if its compute class gets changed.
	if appInstance.Generation != appInstance.Status.ObservedGeneration || needsCalculation(appInstance) {
		if err = calculate(req, appInstance); err != nil {
			return err
		}
	}

	return nil
}

func calculate(req router.Request, appInstance *internalv1.AppInstance) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	if appInstance != nil {
		if err = AddDefaultRegion(req.Ctx, req.Client, appInstance); err != nil {
			return err
		}
	}

	if err = resolveComputeClasses(req, cfg, appInstance); err != nil {
		return err
	}

	return nil
}

func needsCalculation(appInstance *internalv1.AppInstance) bool {
	for _, name := range appInstance.GetAllContainerNames() {
		if _, ok := appInstance.Status.ResolvedOfferings.Containers[name]; !ok {
			return true
		}
	}
	return false
}
