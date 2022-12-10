package builder

import (
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/acorn-io/baaah/pkg/router"
)

func DeployRegistry(req router.Request, resp router.Response) error {
	obj, err := imagesystem.GetRegistryObjects(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	resp.Objects(obj...)
	return nil
}
