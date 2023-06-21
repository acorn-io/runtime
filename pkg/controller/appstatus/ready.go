package appstatus

import (
	"errors"
	"strings"

	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/condition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func ReadyStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	app.Status.Ready = false
	cond := condition.Setter(app, resp, v1.AppInstanceConditionReady)

	var (
		errs             []error
		transitioning    = sets.NewString()
		conditionSuccess = true
	)
	for _, condition := range app.Status.Conditions {
		if condition.Type == v1.AppInstanceConditionReady {
			continue
		}

		if condition.Status == metav1.ConditionFalse {
			errs = append(errs, errors.New(condition.Message))
		} else if condition.Status == metav1.ConditionUnknown && condition.Message != "" {
			transitioning.Insert(condition.Message)
		}

		if !condition.Success {
			conditionSuccess = false
		}
	}

	if len(errs) > 0 {
		if transitioning.Len() > 0 {
			errs = append(errs, errors.New(strings.Join(transitioning.List(), ", ")))
		}
		cond.Error(merr.NewErrors(errs...))
		return nil
	}

	if transitioning.Len() > 0 {
		cond.Unknown(strings.Join(transitioning.List(), ", "))
		return nil
	}

	app.Status.Ready = app.Status.AppImage.ID != "" &&
		app.Generation == app.Status.ObservedGeneration &&
		conditionSuccess
	if app.Status.Ready {
		cond.Success()
	} else {
		cond.Unknown("Not ready")
	}
	return nil
}
