package config

import (
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/baaah/pkg/router"
)

// HandleAutoUpgradeInterval resets the timer for auto-upgrade sync interval as it changes in the acorn config
func HandleAutoUpgradeInterval(router.Request, router.Response) error {
	autoupgrade.Sync()
	return nil
}
