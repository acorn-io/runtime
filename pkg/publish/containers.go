package publish

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/ports"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Containers(app *v1.AppInstance) ([]kclient.Object, error) {
	if app.Spec.Stop != nil && *app.Spec.Stop {
		return nil, nil
	}

	portSet, err := ports.NewForServiceLBPublish(app)
	if err != nil {
		return nil, err
	}

	return ports.ToContainerServices(app, true, app.Status.Namespace, portSet), nil
}
