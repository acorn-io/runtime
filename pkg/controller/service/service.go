package service

import (
	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/runtime/pkg/publish"
	"github.com/acorn-io/runtime/pkg/services"
)

func RenderServices(req router.Request, resp router.Response) error {
	svcInstance := req.Object.(*v1.ServiceInstance)

	// reset, should get repopulated
	svcInstance.Status.Endpoints = nil

	var svcList []*v1.ServiceInstance
	http2Ports := ports.ByProtocol(svcInstance.Spec.Ports, true, v1.ProtocolHTTP2)
	otherPorts := ports.ByProtocol(svcInstance.Spec.Ports, false, v1.ProtocolHTTP2)
	if len(http2Ports) > 0 {
		svcCopy := svcInstance.DeepCopy()
		svcCopy.Spec.Ports = http2Ports
		// append uuid to reduce the chance of clash
		svcCopy.Name = name.SafeConcatName(svcInstance.Name, string(v1.ProtocolHTTP2), string(svcInstance.UID))
		if svcCopy.Spec.Annotations == nil {
			svcCopy.Spec.Annotations = map[string]string{}
		}
		svcCopy.Spec.Annotations["traefik.ingress.kubernetes.io/service.serversscheme"] = "h2c"
		svcList = append(svcList, svcCopy)
	}
	svcCopy := svcInstance.DeepCopy()
	svcCopy.Spec.Ports = otherPorts
	svcList = append(svcList, svcCopy)

	for _, svc := range svcList {
		objs, _, err := services.ToK8sService(req, svc)
		if err != nil {
			return err
		}
		resp.Objects(objs...)
		svcInstance.Status.HasService = svcInstance.Status.HasService || len(objs) > 0

		objs, err = publish.ServiceLoadBalancer(req, svc)
		if err != nil {
			return err
		}
		resp.Objects(objs...)
		svcInstance.Status.HasService = svcInstance.Status.HasService || len(objs) > 0

		objs, err = publish.Ingress(req, svc)
		if err != nil {
			return err
		}
		resp.Objects(objs...)

		// Copy all modifications made by the above publish.Ingress call
		svcInstance.Status.Endpoints = append(svcInstance.Status.Endpoints, svc.Status.Endpoints...)
	}
	svcInstance.Status.Conditions = svcCopy.Status.Conditions

	resp.Objects(svcInstance)
	return nil
}
