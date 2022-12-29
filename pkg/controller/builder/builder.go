package builder

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/acorn-io/acorn/pkg/publish"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	registryDNS, err := imagesystem.GetClusterInternalRegistryDNSName(req.Ctx, req.Client)
	if err != nil {
		return "", "", nil, err
	}

	var forNamespace string
	if *cfg.BuilderPerNamespace {
		forNamespace = builder.Namespace
	}

	objs := imagesystem.BuilderObjects(name, system.ImagesNamespace, forNamespace, system.DefaultImage(), pubKey, privKey, builder.Status.UUID, registryDNS)

	if *cfg.PublishBuilders {
		ing, err := getIngress(req, name)
		if err != nil {
			return "", "", nil, err
		}
		objs = append(objs, ing...)
	}

	if *cfg.BuilderPerNamespace {
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

	if *cfg.BuilderPerNamespace {
		if builder.Status.UUID == "" {
			builder.Status.UUID = uuid.New().String()
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
		ing, ok := obj.(*networkingv1.Ingress)
		if ok {
			if len(ing.Spec.TLS) > 0 {
				builder.Status.Endpoint = "https://" + ing.Spec.Rules[0].Host
			} else {
				builder.Status.Endpoint = "http://" + ing.Spec.Rules[0].Host
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

func getIngress(req router.Request, name string) ([]kclient.Object, error) {
	return publish.Ingress(req, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: system.ImagesNamespace,
		},
		Spec: v1.AppInstanceSpec{},
		Status: v1.AppInstanceStatus{
			Namespace: system.ImagesNamespace,
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					name: {
						Ports: v1.Ports{
							{
								Port:     8080,
								Protocol: v1.ProtocolHTTP,
								Publish:  true,
							},
						},
					},
				},
			},
		},
	})
}
