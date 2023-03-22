package service

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/publish"
	"github.com/acorn-io/acorn/pkg/services"
	"github.com/acorn-io/baaah/pkg/router"
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

	objs, err = publish.ServiceLoadBalancer(svcInstance)
	if err != nil {
		return err
	}
	resp.Objects(objs...)

	objs, err = publish.Ingress(req, svcInstance)
	if err != nil {
		return err
	}
	resp.Objects(objs...)

	return nil
}
