package validate

import (
	"fmt"
	"time"
)

// AutoUpgradeInterval checks that the supplied val can be parsed as a duration and is greater than the system minimum (15 seconds).
func AutoUpgradeInterval(val string) (time.Duration, error) {
	newDur, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("auto-upgrade-interval's value \"%v\" is invalid. Must be an interval with a time unit like  \"5m\"", val)
	}

	if newDur < 15*time.Second {
		return 0, fmt.Errorf("auto-upgrade-interval %v is too small. Must be at least 15 seconds", val)
	}

	return newDur, nil
}
