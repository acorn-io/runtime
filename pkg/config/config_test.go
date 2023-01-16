package config

import (
	"context"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
)

func TestAcornDNSDisabledNoLookupsHappen(t *testing.T) {
	s := "not exactly disabled, but any string that doesn't equal" +
		" auto or enabled should be treated as disabled"
	_ = complete(context.Background(), &apiv1.Config{
		AcornDNS: &s,
	}, nil)
	// if a lookup is going to happen this method would panic as the getter is nil
}
