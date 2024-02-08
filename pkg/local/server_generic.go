//go:build !linux

package local

import (
	"context"
	"fmt"
)

func ServerRun(context.Context) error {
	return fmt.Errorf("only supported on linux")
}
