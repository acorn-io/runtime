package service

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/publish"
	"github.com/acorn-io/acorn/pkg/services"
	"github.com/acorn-io/baaah/pkg/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func getAppForService(req router.Request) (*v1.AppInstance, error) {
	svcInstance := req.Object.(*v1.ServiceInstance)
	app := &v1.AppInstance{}
	err := req.Get(app, svcInstance.Labels[labels.AcornAppNamespace], svcInstance.Labels[labels.AcornAppName])
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	return app, err
}

func RenderServices(req router.Request, resp router.Response) error {
	svcInstance := req.Object.(*v1.ServiceInstance)
	app, err := getAppForService(req)
	if app == nil || err != nil {
		return err
	}

	if app.Status.AppImage.ID == "" {
		return nil
	}

	objs, _, err := services.ToK8sService(req, app, svcInstance)
	if err != nil {
		return err
	}
	resp.Objects(objs...)

	objs, err = publish.ServiceLoadBalancer(app, svcInstance)
	if err != nil {
		return err
	}
	resp.Objects(objs...)

	objs, err = publish.Ingress(req, app, svcInstance)
	if err != nil {
		return err
	}
	resp.Objects(objs...)

	return nil
}
