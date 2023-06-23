package devsession

import (
	"encoding/json"
	"hash/fnv"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/publicname"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func ExpireDevSession(req router.Request, resp router.Response) error {
	if delay := expired(req.Object.(*v1.DevSessionInstance)); delay < 0 && req.Object.GetDeletionTimestamp().IsZero() {
		// Don't delete devsession when the app is removing because this latest devsession might have the info in
		// it to properly remove the object
		app := &v1.AppInstance{}
		if err := req.Get(app, req.Namespace, req.Name); err == nil && !app.DeletionTimestamp.IsZero() {
			return nil
		}
		return req.Client.Delete(req.Ctx, req.Object)
	} else if delay >= 0 {
		resp.RetryAfter(delay)
	}
	return nil
}

func OverlayDevSession(next router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		oldGeneration, err := updateAppForDevSession(req, resp)
		if err != nil {
			return err
		}
		err = next.Handle(req, resp)
		if err != nil {
			return err
		}
		if oldGeneration > 0 {
			app := req.Object.(*v1.AppInstance)
			if app.Generation == app.Status.ObservedGeneration {
				app.Status.ObservedGeneration = oldGeneration
			}
			app.Generation = oldGeneration
		}
		return nil
	})
}

func getNewGeneration(devSession *v1.DevSessionInstance) int64 {
	data, _ := json.Marshal(devSession.Spec.SpecOverride)
	h := fnv.New64a()
	_, _ = h.Write(data)
	v := int64(h.Sum64())
	if v < 0 {
		v = 0 - v
	}
	return v
}

func updateAppForDevSession(req router.Request, resp router.Response) (int64, error) {
	if req.Object == nil {
		return 0, nil
	}

	app := req.Object.(*v1.AppInstance)
	devSession := &v1.DevSessionInstance{}

	if err := req.Get(devSession, app.Namespace, publicname.Get(app)); apierror.IsNotFound(err) {
		app.Status.DevSession = nil
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	generation := int64(0)
	app.Status.DevSession = &devSession.Spec
	if devSession.Spec.SpecOverride != nil {
		generation = app.Generation
		app.Generation = getNewGeneration(devSession)
		app.Spec = *devSession.Spec.SpecOverride
		// If already in sync, keep in sync
		if app.Status.ObservedGeneration == generation {
			app.Status.ObservedGeneration = app.Generation
		}
	}

	return generation, nil
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
