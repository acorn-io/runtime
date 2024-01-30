package apps

import (
	"context"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	mtypes "github.com/acorn-io/mink/pkg/types"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newAppInstanceStrategy(c kclient.WithWatch) strategy.CompleteStrategy {
	return &appInstanceStrategy{
		CompleteStrategy: remote.NewRemote(&v1.AppInstance{}, c),
		c:                c,
	}
}

type appInstanceStrategy struct {
	strategy.CompleteStrategy
	c kclient.WithWatch
}

func (s *appInstanceStrategy) Update(ctx context.Context, obj mtypes.Object) (mtypes.Object, error) {
	// Get the existing object
	var existing v1.AppInstance
	if err := s.c.Get(ctx, kclient.ObjectKeyFromObject(obj), &existing); err != nil {
		return nil, err
	}

	// Ensure that AppInstanceStatus fields not surfaced by AppStatus are preserved.
	// Note: This is necessary because obj has been translated from an apiv1.App before being passed to this method,
	// and since the AppStatus is a subset of the AppInstanceStatus, the App
	if existing.Status.Scheduling != nil {
		obj.(*v1.AppInstance).Status.Scheduling = existing.Status.Scheduling
	}

	return s.CompleteStrategy.Update(ctx, obj)
}
