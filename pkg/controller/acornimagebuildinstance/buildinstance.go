package acornimagebuildinstance

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
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
