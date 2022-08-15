package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/expose"
	"github.com/acorn-io/acorn/pkg/publish"
	"github.com/acorn-io/baaah/pkg/router"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addPublish(req router.Request, app *v1.AppInstance) ([]kclient.Object, error) {
	objs, err := publish.Containers(app)
	if err != nil {
		return nil, err
	}

	ingresses, err := publish.Ingress(req, app)
	if err != nil {
		return nil, err
	}
	objs = append(objs, ingresses...)
	return objs, nil
}

func addExpose(req router.Request, app *v1.AppInstance) ([]kclient.Object, error) {
	objs, err := expose.Links(req, app)
	if err != nil {
		return nil, err
	}

	containers, err := expose.Containers(app)
	if err != nil {
		return nil, err
	}
	objs = append(objs, containers...)

	acorns, err := expose.Acorns(req, app)
	if err != nil {
		return nil, err
	}
	objs = append(objs, acorns...)
	return objs, nil
}
