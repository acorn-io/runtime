package appstatus

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func SetParentStatus(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)

	parentName, ok := app.GetLabels()[labels.AcornParentAcornName]
	if !ok {
		return nil
	}

	var parent *v1.AppInstance
	if err := req.Client.Get(req.Ctx, router.Key(app.Namespace, parentName), parent); err != nil {
		if apierrors.IsNotFound(err) {
			// Parent may be gone already, so we ignore this
			return nil
		}
		return err
	}

	if parent.Status.Parents != nil {
		// Copy the parent's parent images
		app.Status.Parents = make(map[string]v1.ParentStatus, len(parent.Status.Parents)+1)
		for k, v := range parent.Status.Parents {
			app.Status.Parents[k] = v
		}

		// Add the parent's image
		app.Status.Parents[parentName] = v1.ParentStatus{
			ImageName:   parent.Status.AppImage.Name,
			ImageDigest: parent.Status.AppImage.Digest,
		}
	}

	return nil
}
