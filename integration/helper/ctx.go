package helper

import (
	"context"
	"testing"
)

func GetCTX(t *testing.T) context.Context {
	deadline, ok := t.Deadline()
	if !ok {
		return context.Background()
	}
	ctx, _ := context.WithDeadline(context.Background(), deadline)
	return ctx
}
