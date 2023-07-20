package autoupgrade

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
)

func SetOriginalImageAnnotation(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)

	if app.Status.AppImage.Name != "" {
		if app.Annotations == nil {
			app.Annotations = make(map[string]string)
		}

		app.Annotations[labels.AcornOriginalImage] = app.Status.AppImage.Name
		resp.Objects(app)
	}
	return nil
}
