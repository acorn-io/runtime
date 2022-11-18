package config

import (
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/baaah/pkg/router"
)

// HandleAutoUpgradeInterval resets the ticker for auto-upgrade sync interval as it changes in the acorn config
func HandleAutoUpgradeInterval(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	if cfg.AutoUpgradeInterval != nil {
		err := autoupgrade.UpdateInterval(*cfg.AutoUpgradeInterval)
		return err
	}

	return nil
}
