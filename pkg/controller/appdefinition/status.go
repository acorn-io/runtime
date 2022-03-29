package appdefinition

import (
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

func AppStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	deps := &appsv1.DeploymentList{}

	err := req.Client.List(deps, &meta.ListOptions{
		Namespace: app.Status.Namespace,
		Selector: klabels.SelectorFromSet(map[string]string{
			labels.HerdAppName: app.Name,
		}),
	})
	if err != nil {
		return err
	}

	container := map[string]v1.ContainerStatus{}
	for _, dep := range deps.Items {
		status := container[dep.Labels[labels.HerdContainerName]]
		status.Ready = dep.Status.ReadyReplicas
		status.ReadyDesired = dep.Status.Replicas
		status.UpToDate = dep.Status.UpdatedReplicas
		container[labels.HerdContainerName] = status
	}
	app.Status.ContainerStatus = container

	resp.Objects(app)
	return nil
}
