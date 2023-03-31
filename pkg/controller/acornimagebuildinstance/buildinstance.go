package acornimagebuildinstance

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/controller/defaults"
	"github.com/acorn-io/baaah/pkg/router"
)

func MarkRecorded(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}
	if !*cfg.RecordBuilds {
		return nil
	}
	req.Object.(*v1.AcornImageBuildInstance).Status.Recorded = true
	return nil
}

func SetRegion(req router.Request, _ router.Response) error {
	return defaults.AddDefaultRegion(req.Ctx, req.Client, req.Object.(*v1.AcornImageBuildInstance))
}
