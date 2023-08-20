package builder

import (
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/system"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func createBuilderObjects(req router.Request, resp router.Response) (string, string, []kclient.Object, error) {
	builder := req.Object.(*v1.BuilderInstance)

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return "", "", nil, err
	}

	name, err := imagesystem.GetBuilderDeploymentName(req.Ctx, req.Client, builder.Name, builder.Namespace)
	if err != nil {
		return "", "", nil, err
	}

	pubKey, privKey, err := imagesystem.GetBuilderKeys(req.Ctx, req.Client, system.ImagesNamespace, name)
	if err != nil {
		return "", "", nil, err
	}

	depotToken, depotProjectId, err := getDepotKey(req.Ctx, req.Client, builder.Namespace)
	if err != nil {
		return "", "", nil, err
	}

	registryDNS, err := imagesystem.GetClusterInternalRegistryDNSName(req.Ctx, req.Client)
	if err != nil {
		return "", "", nil, err
	}

	var forNamespace string
	if *cfg.BuilderPerProject {
		forNamespace = builder.Namespace
	}

	objs := imagesystem.BuilderObjects(name, system.ImagesNamespace, forNamespace, system.DefaultImage(),
		pubKey, privKey, depotToken, depotProjectId, builder.Status.UUID, registryDNS, cfg)

	if *cfg.BuilderPerProject {
		resp.Objects(objs...)
		return name, pubKey, objs, nil
	}

	return name, pubKey, objs, apply.New(req.Client).Ensure(req.Ctx, objs...)
}

func DeployBuilder(req router.Request, resp router.Response) error {
	builder := req.Object.(*v1.BuilderInstance)
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	if *cfg.BuilderPerProject {
		if builder.Status.UUID == "" {
			builder.Status.UUID = string(builder.UID)
		}
	} else {
		builder.Status.UUID = ""
	}

	serviceName, pubKey, objs, err := createBuilderObjects(req, resp)
	if err != nil {
		return err
	}

	builder.Status.ObservedGeneration = builder.Generation
	builder.Status.PublicKey = pubKey
	builder.Status.Endpoint = ""
	builder.Status.ServiceName = serviceName

	for _, obj := range objs {
		svc, ok := obj.(*v1.ServiceInstance)
		if ok && *cfg.PublishBuilders {
			if len(svc.Status.Endpoints) == 0 {
				existing := &v1.ServiceInstance{}
				err := req.Get(existing, svc.Namespace, svc.Name)
				if err == nil {
					svc = existing
				}
			}
			if len(svc.Status.Endpoints) > 0 {
				if svc.Status.Endpoints[0].PublishProtocol == v1.PublishProtocolHTTPS {
					builder.Status.Endpoint = "https://" + svc.Status.Endpoints[0].Address
				} else {
					builder.Status.Endpoint = "http://" + svc.Status.Endpoints[0].Address
				}
			}
			continue
		}

		dep, ok := obj.(*appsv1.Deployment)
		if !ok {
			continue
		}

		newDep := &appsv1.Deployment{}
		err := req.Get(newDep, dep.Namespace, dep.Name)
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return err
		}

		builder.Status.Ready = newDep.Status.ReadyReplicas > 0
	}

	return nil
}
