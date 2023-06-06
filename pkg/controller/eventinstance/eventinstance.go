package eventinstance

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/baaah/pkg/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultTTL = 168 * time.Hour // 7 days

func GCExpired() router.HandlerFunc {
	// last stores pre-parsed TTLs from the configuration.
	last := new(struct {
		mu  sync.Mutex
		raw string
		ttl time.Duration
	})

	return handler{
		getTTL: func(ctx context.Context, getter kclient.Reader) (time.Duration, error) {
			cfg, err := config.Get(ctx, getter)
			if err != nil {
				return 0, err
			}

			cfgTTL := cfg.EventTTL
			if cfgTTL == nil || *cfgTTL == "" {
				return defaultTTL, nil
			}

			last.mu.Lock()
			defer last.mu.Unlock()

			if last.raw != *cfgTTL {
				// This is a new TTL, parse and memoize
				ttl, err := time.ParseDuration(*cfgTTL)
				if err != nil {
					return 0, err
				}

				last.raw, last.ttl = *cfgTTL, ttl
			}

			return last.ttl, nil
		},
	}.gcExpired
}

// ttlFunc is a function that returns the TTL to use for event expiration.
type ttlFunc func(context.Context, kclient.Reader) (time.Duration, error)

type handler struct {
	getTTL ttlFunc
}

func (h handler) gcExpired(req router.Request, resp router.Response) error {
	e := req.Object.(*v1.EventInstance)

	// Get the currently configured TTL
	ttl, err := h.getTTL(req.Ctx, req.Client)
	if err != nil {
		return fmt.Errorf("failed to get event ttl: %w", err)
	}

	// Check expiration
	if now, expiration := time.Now(), e.Observed.Add(ttl); now.Before(expiration) {
		// Still fresh, wait until expiration
		resp.RetryAfter(time.Until(expiration))
		return nil
	}

	// Expired, delete the event
	if err := req.Client.Delete(req.Ctx, e, kclient.Preconditions{
		// Adding these preconditions prevents us from deleting an event based on old information.
		// e.g. The observed time has been updated and the event is no longer expired.
		UID:             &e.UID,
		ResourceVersion: &e.ResourceVersion,
	}); err != nil && !apierrors.IsNotFound(err) {
		// Assume any error other than not found is transient, return error to requeue w/ backoff
		return err
	}

	return nil
}
