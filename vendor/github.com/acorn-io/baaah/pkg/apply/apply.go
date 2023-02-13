package apply

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultNamespace = "default"
)

var (
	// validOwnerChange is a mapping of old to new subcontext
	// values that are allowed to change.  The key is in the format of
	// "oldValue => newValue"
	// subcontext and the value is the old subcontext name.
	validOwnerChange = map[string]bool{}
)

func AddValidOwnerChange(oldSubcontext, newSubContext string) {
	validOwnerChange[fmt.Sprintf("%s => %s", oldSubcontext, newSubContext)] = true
}

type Apply interface {
	Ensure(ctx context.Context, obj ...kclient.Object) error
	Apply(ctx context.Context, owner kclient.Object, objs ...kclient.Object) error
	WithOwnerSubContext(ownerSubContext string) Apply
	WithNamespace(ns string) Apply
	WithPruneGVKs(gvks ...schema.GroupVersionKind) Apply
	WithPruneTypes(gvks ...kclient.Object) Apply
	WithNoPrune() Apply

	FindOwner(ctx context.Context, obj kclient.Object) (kclient.Object, error)
	PurgeOrphan(ctx context.Context, obj kclient.Object) error
}

func Ensure(ctx context.Context, client kclient.Client, obj ...kclient.Object) error {
	return New(client).Ensure(ctx, obj...)
}

func New(c kclient.Client) Apply {
	return &apply{
		client:           c,
		reconcilers:      defaultReconcilers,
		defaultNamespace: defaultNamespace,
	}
}
