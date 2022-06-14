package cluster

import (
	"encoding/json"
	"net/http"

	uiv1 "github.com/acorn-io/acorn/pkg/apis/ui.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/install"
	"github.com/acorn-io/acorn/pkg/install/progress"
	"github.com/rancher/apiserver/pkg/types"
)

func Install(rw http.ResponseWriter, req *http.Request) {
	apiContext := types.GetAPIContext(req.Context())
	installOpts := &uiv1.Install{}
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(installOpts); err != nil {
		apiContext.WriteError(err)
		return
	}

	var image = install.DefaultImage()
	if installOpts.Image != "" {
		image = installOpts.Image
	}

	err := install.Install(req.Context(), image, &install.Options{
		Progress:           progress.NewStream(rw),
		Config:             installOpts.Config,
		Mode:               installOpts.Mode,
		APIServerReplicas:  installOpts.APIServerReplicas,
		ControllerReplicas: installOpts.ControllerReplicas,
	})
	if err != nil {
		apiContext.WriteError(err)
		return
	}
}
