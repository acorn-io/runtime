package defaults

import (
	"context"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/z"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addDefaultVolumeSize(ctx context.Context, c client.Client, appInstance *v1.AppInstance) error {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return err
	}

	defaultVolumeSize := z.Pointer(v1.DefaultSize.DeepCopy())

	// If the default volume size is set in the config, use that instead.
	if cfgVolumeSize, err := resource.ParseQuantity(cfg.VolumeSizeDefault); err == nil && cfg.VolumeSizeDefault != "" {
		defaultVolumeSize = &cfgVolumeSize
	}

	appInstance.Status.Defaults.VolumeSize = defaultVolumeSize
	return nil
}
