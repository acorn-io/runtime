package service

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/publish"
	"github.com/acorn-io/runtime/pkg/services"
)

func RenderServices(req router.Request, resp router.Response) error {
	svcInstance := req.Object.(*v1.ServiceInstance)

	// reset, should get repopulated
	svcInstance.Status.Endpoints = nil

	objs, _, err := services.ToK8sService(req, svcInstance)
	if err != nil {
		return err
	}
	resp.Objects(objs...)
	svcInstance.Status.HasService = len(objs) > 0

	objs, err = publish.ServiceLoadBalancer(req, svcInstance)
	if err != nil {
		return err
	}
	resp.Objects(objs...)
	svcInstance.Status.HasService = svcInstance.Status.HasService || len(objs) > 0

	objs, err = publish.Ingress(req, svcInstance)
	if err != nil {
		return err
	}
	resp.Objects(objs...)

	resp.Objects(svcInstance)
	return nil
}
