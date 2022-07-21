package expose

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/ports"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Containers(app *v1.AppInstance) ([]kclient.Object, error) {
	portSet, err := ports.New(app)
	if err != nil {
		return nil, err
	}

	return ports.ToContainerServices(app, false, app.Status.Namespace, portSet), nil
}
