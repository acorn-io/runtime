package region

import (
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateLocalRegion(_ router.Request, resp router.Response) error {
	resp.Objects(&adminv1.RegionInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
		},
		Spec: adminv1.RegionInstanceSpec{
			Description: "Acorn-generated region for the local cluster",
			AccountName: "local",
			RegionName:  "local",
		},
	})

	return nil
}
