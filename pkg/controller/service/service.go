package service

import (
	"fmt"

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
	httpPorts := ports.ByProtocol(svcInstance.Spec.Ports, true, v1.ProtocolHTTP)
	// If we have both types of ports we need to create a separate service with the annotation
	if len(http2Ports) > 0 && len(httpPorts) > 0 {
		svcCopy := svcInstance.DeepCopy()
		svcCopy.Spec.Ports = ports.ByProtocol(svcInstance.Spec.Ports, false, v1.ProtocolHTTP2) // filter out http2 ports
		svcList = append(svcList, svcCopy)

		svcCopyHttp2 := svcInstance.DeepCopy()
		svcCopyHttp2.Spec.Ports = http2Ports
		svcCopyHttp2.Name = fmt.Sprintf("%s-%s", svcInstance.Name, v1.ProtocolHTTP2)
		if svcCopyHttp2.Spec.Annotations == nil {
			svcCopyHttp2.Spec.Annotations = map[string]string{}
		}
		svcCopyHttp2.Spec.Annotations["traefik.ingress.kubernetes.io/service.serversscheme"] = "h2c"
		svcList = append(svcList, svcCopyHttp2)
	} else if len(http2Ports) > 0 {
		svcCopyHttp2 := svcInstance.DeepCopy()
		if svcCopyHttp2.Spec.Annotations == nil {
			svcCopyHttp2.Spec.Annotations = map[string]string{}
		}
		svcCopyHttp2.Spec.Annotations["traefik.ingress.kubernetes.io/service.serversscheme"] = "h2c"
		svcList = append(svcList, svcCopyHttp2)
	} else {
		svcList = []*v1.ServiceInstance{svcInstance}
	}

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

	resp.Objects(svcInstance)
	return nil
}
