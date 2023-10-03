package defaults

import (
	"context"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/z"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addDefaultVolumeSize(ctx context.Context, c client.Client, appInstance *v1.AppInstance) error {
	// Only set Defaults.VolumeSize once.
	if appInstance.Status.Defaults.VolumeSize != nil {
		return nil
	}
	defaultVolumeSize := z.Pointer(v1.DefaultSize.DeepCopy())
	appInstance.Status.Defaults.VolumeSize = defaultVolumeSize
	return nil
}
