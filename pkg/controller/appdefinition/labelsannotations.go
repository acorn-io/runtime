package appdefinition

import (
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	"golang.org/x/exp/maps"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func AddCommonLabelsAnnotations(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		return h.Handle(req, &labelAnnonResponse{
			app:  req.Object.(*v1.AppInstance),
			req:  req,
			resp: resp,
		})
	})
}

type labelAnnonResponse struct {
	app  *v1.AppInstance
	req  router.Request
	resp router.Response
}

func (d *labelAnnonResponse) RetryAfter(delay time.Duration) {
	d.resp.RetryAfter(delay)
}

func (d *labelAnnonResponse) Objects(objs ...kclient.Object) {
	for _, obj := range objs {
		if len(d.app.Spec.Annotations) > 0 || len(d.app.Status.AppSpec.Annotations) > 0 {
			if obj.GetAnnotations() == nil {
				obj.SetAnnotations(make(map[string]string))
			}
			maps.Copy(obj.GetAnnotations(), d.app.Status.AppSpec.Annotations)
			maps.Copy(obj.GetAnnotations(), d.app.Spec.Annotations)
		}

		if len(d.app.Spec.Labels) > 0 || len(d.app.Status.AppSpec.Labels) > 0 {
			if obj.GetLabels() == nil {
				obj.SetLabels(make(map[string]string))
			}
			maps.Copy(obj.GetLabels(), d.app.Status.AppSpec.Labels)
			maps.Copy(obj.GetLabels(), d.app.Spec.Labels)
		}
	}
	d.resp.Objects(objs...)
}
