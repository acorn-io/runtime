package buildserver

import (
	"context"

	"github.com/acorn-io/baaah/pkg/apply"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteOld(ctx context.Context, c client.Client) error {
	apply := apply.New(c)
	return apply.
		WithOwnerSubContext("acorn-buildkitd").
		WithPruneGVKs(schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Service",
		}, schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		}, schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
		}).
		Apply(ctx, nil)
}
