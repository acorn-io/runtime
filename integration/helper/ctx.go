package helper

import (
	"context"
	"testing"
	"time"
)

func GetCTX(t *testing.T) context.Context {
	ctx := context.Background()
	deadline, ok := t.Deadline()
	if !ok {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		t.Cleanup(func() {
			cancel()
		})
		return ctx
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	t.Cleanup(func() {
		cancel()
	})
	return ctx
}
