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
	app.Status.Summary = v1.CommonSummary{}
	cond := condition.Setter(app, resp, v1.AppInstanceConditionReady)

	var (
		errs             []error
		transitioning    = sets.NewString()
		errorMessages    = sets.NewString()
		conditionSuccess = true
	)
	for _, condition := range app.Status.Conditions {
		if condition.Type == v1.AppInstanceConditionReady {
			continue
		}

		if condition.Status == metav1.ConditionFalse {
			errs = append(errs, errors.New(condition.Message))
			errorMessages.Insert(condition.Message)
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
	} else if transitioning.Len() > 0 {
		cond.Unknown(strings.Join(transitioning.List(), ", "))
		app.Status.Summary.TransitioningMessages = transitioning.List()
	} else {
		app.Status.Ready = app.Status.AppImage.ID != "" &&
			app.Generation == app.Status.ObservedGeneration &&
			conditionSuccess
		if app.Status.Ready {
			cond.Success()
		} else {
			cond.Unknown("Not ready")
		}
	}

	var state string
	if !app.DeletionTimestamp.IsZero() {
		state = "removing"
	} else if app.GetStopped() {
		if app.Status.AppStatus.Stopped {
			state = "stopped"
		} else {
			state = "stopping"
		}
	} else if errorMessages.Len() > 0 {
		app.Status.Summary.ErrorMessages = []string{errorMessages.List()[0]}
		state = "error"
	} else if transitioning.Len() > 0 {
		app.Status.Summary.TransitioningMessages = []string{transitioning.List()[0]}
		state = "provisioning"
	} else if app.Status.Ready {
		if app.Status.AppStatus.Completed {
			state = "completed"
		} else {
			state = "running"
		}
	} else {
		state = "not ready"
	}

	app.Status.Summary.State = state
	return nil
}
