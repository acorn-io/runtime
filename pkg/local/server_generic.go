//go:build !linux

package local

import (
	"context"
	_ "embed"
	"fmt"
)

func ServerRun(ctx context.Context) error {
	return fmt.Errorf("only supported on linux")
}
