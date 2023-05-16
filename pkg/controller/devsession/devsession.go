package devsession

import (
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/baaah/pkg/router"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func ExpireDevSession(req router.Request, resp router.Response) error {
	if delay := expired(req.Object.(*v1.DevSessionInstance)); delay < 0 && req.Object.GetDeletionTimestamp().IsZero() {
		return req.Client.Delete(req.Ctx, req.Object)
	} else if delay >= 0 {
		resp.RetryAfter(delay)
	}
	return nil
}

func OverlayDevSession(next router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		if err := updateAppForDevSession(req, resp); err != nil {
			return err
		}
		return next.Handle(req, resp)
	})
}

func updateAppForDevSession(req router.Request, resp router.Response) error {
	if req.Object == nil {
		return nil
	}

	app := req.Object.(*v1.AppInstance)
	devSession := &v1.DevSessionInstance{}

	if err := req.Get(devSession, app.Namespace, publicname.Get(app)); apierror.IsNotFound(err) {
		app.Status.DevSession = nil
		return nil
	} else if err != nil {
		return err
	}

	app.Status.DevSession = &devSession.Spec
	if devSession.Spec.SpecOverride != nil {
		app.Spec = *devSession.Spec.SpecOverride
	}

	return nil
}

func releaseTime(devSession *v1.DevSessionInstance) time.Time {
	renewTime := devSession.Spec.SessionRenewTime
	if renewTime.IsZero() {
		renewTime = devSession.Spec.SessionStartTime
	}
	return renewTime.Add(time.Duration(devSession.Spec.SessionTimeoutSeconds) * time.Second)
}

func expired(devSession *v1.DevSessionInstance) time.Duration {
	releaseTime := releaseTime(devSession)
	return time.Until(releaseTime)
}
