package resolvedofferings

import (
	"context"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RegionGetterSetter interface {
	metav1.Object
	GetRegion() string
	SetDefaultRegion(string)
}

func AddDefaultRegion(ctx context.Context, c client.Client, obj RegionGetterSetter) error {
	if obj.GetRegion() == "" {
		project := new(v1.ProjectInstance)
		if err := c.Get(ctx, client.ObjectKey{Name: obj.GetNamespace()}, project); err != nil {
			return err
		}

		obj.SetDefaultRegion(project.Status.DefaultRegion)
	} else {
		obj.SetDefaultRegion(obj.GetRegion())
	}

	return nil
}

func SetDefaultRegion(req router.Request, _ router.Response) error {
	return AddDefaultRegion(req.Ctx, req.Client, req.Object.(RegionGetterSetter))
}
