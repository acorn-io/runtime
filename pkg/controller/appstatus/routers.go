package appstatus

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/ports"
)

func (a *appStatusRenderer) readRouters() (err error) {
	existingStatus := a.app.Status.AppStatus.Routers
	// reset state
	a.app.Status.AppStatus.Routers = map[string]v1.RouterStatus{}

	for _, routerName := range typed.SortedKeys(a.app.Status.AppSpec.Routers) {
		s := v1.RouterStatus{
			CommonStatus: v1.CommonStatus{
				Defined:      ports.IsLinked(a.app, routerName),
				LinkOverride: ports.LinkService(a.app, routerName),
				ConfigHash:   existingStatus[routerName].ConfigHash,
			},
			MissingTargets: existingStatus[routerName].MissingTargets,
		}

		s.Ready, s.Defined, err = a.isServiceReady(routerName)
		if err != nil {
			return err
		}

		s.UpToDate = s.Defined

		if len(s.MissingTargets) > 0 {
			s.ErrorMessages = append(s.ErrorMessages, fmt.Sprintf("missing route target [%s]", strings.Join(s.MissingTargets, ",")))
		}

		if s.UpToDate && a.app.GetStopped() {
			s.Ready = true
		}

		a.app.Status.AppStatus.Routers[routerName] = s
	}

	return nil
}

func setRouterMessages(app *v1.AppInstance) {
	for routerName, s := range app.Status.AppStatus.Routers {
		// Not ready if we have any error messages
		if len(s.ErrorMessages) > 0 {
			s.Ready = false
		}

		if s.Ready {
			s.State = "ready"
		} else if s.UpToDate {
			if len(s.ErrorMessages) > 0 {
				s.State = "failing"
			} else {
				s.State = "not ready"
			}
		} else if s.Defined {
			if len(s.ErrorMessages) > 0 {
				s.State = "error"
			} else {
				s.State = "updating"
			}
		} else {
			if len(s.ErrorMessages) > 0 {
				s.State = "error"
			} else {
				s.State = "pending"
			}
		}

		app.Status.AppStatus.Routers[routerName] = s
	}
}
