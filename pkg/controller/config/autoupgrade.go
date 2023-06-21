package config

import (
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
)

// HandleAutoUpgradeInterval resets the timer for auto-upgrade sync interval as it changes in the acorn config
func HandleAutoUpgradeInterval(router.Request, router.Response) error {
	autoupgrade.Sync()
	return nil
}
