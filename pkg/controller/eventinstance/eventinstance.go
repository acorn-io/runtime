package eventinstance

import (
	"context"
	"fmt"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"golang.org/x/sync/semaphore"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultTTL = 7 * 24 * time.Hour

func GCExpired() router.HandlerFunc {
	sem := semaphore.NewWeighted(1)

	// parsed stores pre-parsed TTLs from the configuration.
	var parsed struct {
		raw string
		ttl time.Duration
	}

	return handler{
		getTTL: func(
			ctx context.Context,
			getter kclient.Reader,
		) (time.Duration, error) {
			cfg, err := config.Get(ctx, getter)
			if err != nil {
				return 0, err
			}

			cfgTTL := cfg.EventTTL
			if cfgTTL == nil || *cfgTTL == "" {
				return defaultTTL, nil
			}

			if err := sem.Acquire(ctx, 1); err != nil {
				return 0, fmt.Errorf("failed to acquire ttl semaphore: %w", err)
			}
			defer sem.Release(1)

			if parsed.raw != *cfgTTL {
				// This is a new TTL, parse and memoize
				ttl, err := time.ParseDuration(*cfgTTL)
				if err != nil {
					return 0, err
				}

				parsed.raw, parsed.ttl = *cfgTTL, ttl
			}

			return parsed.ttl, nil
		},
	}.gcExpired
}

// GCable describes types that can be GCed by the router.HandlerFunc returned by GCExpired.
type GCable interface {
	// GetObserved returns the time of the initial observation.
	GetObserved() v1.MicroTime
}

type handler struct {
	// getTTL returns the TTL to use for event expiration.
	getTTL func(
		context.Context,
		kclient.Reader,
	) (time.Duration, error)
}

func (h handler) gcExpired(
	req router.Request,
	resp router.Response,
) error {
	e := req.Object

	// Get the currently configured TTL
	ttl, err := h.getTTL(req.Ctx, req.Client)
	if err != nil {
		return fmt.Errorf("failed to get event ttl: %w", err)
	}

	// Check expiration
	if now, expiration := time.Now(), e.(GCable).GetObserved().Add(ttl); now.Before(expiration) {
		// Still fresh, wait until expiration
		resp.RetryAfter(time.Until(expiration))
		return nil
	}

	// Expired, delete the event
	if err := req.Client.Delete(req.Ctx, req.Object, kclient.Preconditions{
		// Adding these preconditions prevents us from deleting an event based on old information.
		// e.g. The observed time has been updated and the event is no longer expired.
		UID:             ptr(e.GetUID()),
		ResourceVersion: ptr(e.GetResourceVersion()),
	}); err != nil && !apierrors.IsNotFound(err) {
		// Assume any error other than not found is transient, return error to requeue w/ backoff
		return err
	}

	return nil
}

func ptr[T any](t T) *T {
	return &t
}
