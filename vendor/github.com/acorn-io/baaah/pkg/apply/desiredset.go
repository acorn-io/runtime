package apply

import (
	"context"

	"github.com/acorn-io/baaah/pkg/apply/objectset"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// reconciler return false if it did not handle this object
type reconciler func(oldObj kclient.Object, newObj kclient.Object) (bool, error)

type apply struct {
	ctx              context.Context
	client           kclient.Client
	defaultNamespace string
	listerNamespace  string
	pruneTypes       map[schema.GroupVersionKind]bool
	pruneObjects     []kclient.Object
	reconcilers      map[schema.GroupVersionKind]reconciler
	ownerSubContext  string
	owner            kclient.Object
	ownerGVK         schema.GroupVersionKind
	ensure           bool
	noPrune          bool
}

func (a apply) Ensure(ctx context.Context, objs ...kclient.Object) error {
	a.ensure = true
	a.owner = nil
	a.ownerSubContext = ""
	return a.Apply(ctx, nil, objs...)
}

func (a apply) Apply(ctx context.Context, owner kclient.Object, objs ...kclient.Object) error {
	var newPruneGVKs []schema.GroupVersionKind
	for _, pruneObject := range a.pruneObjects {
		gvk, err := apiutil.GVKForObject(pruneObject, a.client.Scheme())
		if err != nil {
			return err
		}
		newPruneGVKs = append(newPruneGVKs, gvk)
	}

	a = a.withPruneGVKs(newPruneGVKs...)
	a.ctx = ctx
	a.owner = owner
	if owner != nil {
		gvk, err := apiutil.GVKForObject(a.owner, a.client.Scheme())
		if err != nil {
			return err
		}
		a.ownerGVK = gvk
	}
	os, err := objectset.NewObjectSet(a.client.Scheme(), objs...)
	if err != nil {
		return err
	}
	return a.apply(os)
}

func (a apply) WithNoPrune() Apply {
	a.noPrune = true
	return a
}

func (a apply) WithPruneTypes(objs ...kclient.Object) Apply {
	a.pruneObjects = append(a.pruneObjects, objs...)
	return a
}

// WithPruneGVKs uses a known listing of existing gvks to modify the the prune types to allow for deletion of objects
func (a apply) WithPruneGVKs(gvks ...schema.GroupVersionKind) Apply {
	return a.withPruneGVKs(gvks...)
}

func (a apply) withPruneGVKs(gvks ...schema.GroupVersionKind) apply {
	pruneTypes := make(map[schema.GroupVersionKind]bool, len(gvks))
	for k, v := range a.pruneTypes {
		pruneTypes[k] = v
	}
	for _, gvk := range gvks {
		pruneTypes[gvk] = true
	}
	a.pruneTypes = pruneTypes
	return a
}

func (a apply) WithNamespace(ns string) Apply {
	a.listerNamespace = ns
	a.defaultNamespace = ns
	return a
}

func (a apply) WithOwnerSubContext(ownerSubContext string) Apply {
	a.ownerSubContext = ownerSubContext
	return a
}
