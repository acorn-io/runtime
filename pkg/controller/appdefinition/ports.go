package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/expose"
	"github.com/acorn-io/acorn/pkg/publish"
	"github.com/acorn-io/baaah/pkg/router"
)

func addPublish(req router.Request, app *v1.AppInstance, resp router.Response) error {
	objs, err := publish.Containers(app)
	if err != nil {
		return err
	}
	resp.Objects(objs...)

	objs, err = publish.Ingress(req, app)
	if err != nil {
		return err
	}
	resp.Objects(objs...)
	return nil
}

func addExpose(req router.Request, app *v1.AppInstance, resp router.Response) error {
	links, err := expose.Links(req, app)
	if err != nil {
		return err
	}
	resp.Objects(links...)

	objs, err := expose.Containers(app)
	if err != nil {
		return err
	}
	resp.Objects(objs...)

	objs, err = expose.Acorns(req, app)
	if err != nil {
		return err
	}
	resp.Objects(objs...)
	return nil
}
