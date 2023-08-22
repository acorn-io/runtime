package event

import (
	"context"
	"errors"
	"sort"
	"time"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/sirupsen/logrus"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Truncate truncates the set of stored account events to a maximum of limit once every period.
func Truncate(ctx context.Context, client kclient.Client, period time.Duration, limit int) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logrus.Debug("truncating eventinstances")
			var events internalv1.EventInstanceList
			if err := client.List(ctx, &events); err != nil {
				logrus.WithError(err).Warn("failed to truncate events, error listing events")
				continue
			}

			if len(events.Items) <= limit {
				// Nothing to do
				continue
			}

			// Sort events by observed so that the newest events are first
			sort.Slice(events.Items, func(i, j int) bool {
				return events.Items[i].Observed.After(events.Items[j].Observed.Time)
			})

			var errs []error
			for _, event := range events.Items[limit:] {
				if err := kclient.IgnoreNotFound(client.Delete(ctx, &event)); err != nil {
					errs = append(errs, err)
				}
			}

			if err := errors.Join(errs...); err != nil {
				logrus.WithError(err).Warn("failed to truncate events, error deleting events")
			}
		}
	}
}
