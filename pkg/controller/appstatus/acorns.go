package appstatus

import (
	"strconv"
	"strings"

	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/google/go-containerregistry/pkg/name"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *appStatusRenderer) readAcorns(tag name.Reference) error {
	// reset state
	a.app.Status.AppStatus.Acorns = map[string]v1.AcornStatus{}

	for acornName, acornDef := range a.app.Status.AppSpec.Acorns {
		s := v1.AcornStatus{
			CommonStatus: v1.CommonStatus{
				Defined:      ports.IsLinked(a.app, acornName),
				LinkOverride: ports.LinkService(a.app, acornName),
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
		err := a.c.Get(a.ctx, router.Key(a.app.Namespace, name2.SafeHashConcatName(a.app.Name, acornName)), acorn)
		if apierrors.IsNotFound(err) {
			a.app.Status.AppStatus.Acorns[acornName] = s
			continue
		} else if err != nil {
			return err
		}

		s.Defined = true
		var image string
		if _, isPattern := autoupgrade.AutoUpgradePattern(acornDef.Image); isPattern {
			image = acornDef.Image
		} else if tag != nil {
			if strings.HasPrefix(acornDef.Image, "sha256:") {
				image = strings.TrimPrefix(acornDef.Image, "sha256:")
			} else {
				image = images.ResolveTag(tag, acornDef.Image)
			}
		}
		s.UpToDate = acorn.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation)) &&
			image != "" &&
			image == acorn.Spec.Image &&
			acorn.Status.AppImage.Digest == acorn.Status.Staged.AppImage.Digest
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
