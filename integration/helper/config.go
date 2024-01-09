package helper

import (
	"context"
	"testing"

	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/z"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func EnableFeatureWithRestore(t *testing.T, ctx context.Context, kclient kclient.WithWatch, feature string) {
	t.Helper()

	// enable feature in acorn config
	cfg, err := config.Get(ctx, kclient)
	if err != nil {
		t.Fatal(err)
	}

	featureStateOriginal := cfg.Features[feature]

	if cfg.Features == nil {
		cfg.Features = map[string]bool{}
	}
	cfg.Features[feature] = true

	t.Cleanup(func() {
		// Reset feature state to original value (especially heplful when testing locally)
		cfg.Features = map[string]bool{
			feature: featureStateOriginal,
		}

		err = config.Set(ctx, kclient, cfg)
		if err != nil {
			t.Fatal(err)
		}
	})

	err = config.Set(ctx, kclient, cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func SetIgnoreResourceRequirementsWithRestore(t *testing.T, ctx context.Context, kclient kclient.WithWatch) {
	t.Helper()

	cfg, err := config.Get(ctx, kclient)
	if err != nil {
		t.Fatal(err)
	}

	state := z.Dereference(cfg.IgnoreResourceRequirements)

	cfg.IgnoreResourceRequirements = z.Pointer(true)

	t.Cleanup(func() {
		cfg.IgnoreResourceRequirements = z.Pointer(state)

		err = config.Set(ctx, kclient, cfg)
		if err != nil {
			t.Fatal(err)
		}
	})

	err = config.Set(ctx, kclient, cfg)
	if err != nil {
		t.Fatal(err)
	}
}
