package appstatus

import (
	"strconv"

	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/runtime/pkg/publicname"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *appStatusRenderer) readAcorns() error {
	// reset state
	a.app.Status.AppStatus.Acorns = map[string]v1.AcornStatus{}

	for acornName, acornDef := range a.app.Status.AppSpec.Acorns {
		hash, err := configHash(acornDef)
		if err != nil {
			return err
		}

		s := v1.AcornStatus{
			CommonStatus: v1.CommonStatus{
				Defined:      ports.IsLinked(a.app, acornName),
				LinkOverride: ports.LinkService(a.app, acornName),
				ConfigHash:   hash,
			},
		}

		if s.LinkOverride != "" {
			var err error
			s.UpToDate = true
			s.Ready, _, err = a.isServiceReady(acornName)
			if err != nil {
				return err
			}
			a.app.Status.AppStatus.Acorns[acornName] = s
			continue
		}

		acorn := &v1.AppInstance{}
		err = a.c.Get(a.ctx, router.Key(a.app.Namespace, name2.SafeHashConcatName(a.app.Name, acornName)), acorn)
		if apierrors.IsNotFound(err) {
			a.app.Status.AppStatus.Acorns[acornName] = s
			continue
		} else if err != nil {
			return err
		}

		if acorn.Status.AppStatus.LoginRequired {
			a.app.Status.AppStatus.LoginRequired = true
		}
		s.Defined = true
		s.UpToDate = acorn.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation)) && acorn.Annotations[labels.AcornConfigHashAnnotation] == hash
		s.Ready = s.UpToDate && acorn.Status.Ready
		s.AcornName = publicname.Get(acorn)

		for _, cond := range acorn.Status.Conditions {
			if cond.Type == v1.AppInstanceConditionReady {
				if cond.Status == metav1.ConditionFalse {
					s.ErrorMessages = append(s.ErrorMessages, cond.Message)
				} else if cond.Status == metav1.ConditionUnknown {
					s.TransitioningMessages = append(s.TransitioningMessages, cond.Message)
				}
			}
		}

		a.app.Status.AppStatus.Acorns[acornName] = s
	}

	return nil
}

func setAcornMessages(app *v1.AppInstance) {
	for acornName, s := range app.Status.AppStatus.Acorns {
		if s.Ready {
			s.State = "ready"
		} else if s.UpToDate {
			if len(s.ErrorMessages) > 0 {
				s.State = "error"
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

		app.Status.AppStatus.Acorns[acornName] = s
	}
}
