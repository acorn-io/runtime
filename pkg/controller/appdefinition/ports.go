package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/services"
	"github.com/acorn-io/baaah/pkg/router"
)

func addExpose(app *v1.AppInstance, resp router.Response) {
	resp.Objects(services.ToAcornServices(app)...)
}
