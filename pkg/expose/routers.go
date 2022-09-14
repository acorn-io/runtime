package expose

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/ports"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Routers(app *v1.AppInstance) ([]kclient.Object, error) {
	portSet, err := ports.New(app)
	if err != nil {
		return nil, err
	}

	return ports.ToRouterServices(app, app.Status.Namespace, portSet), nil
}
