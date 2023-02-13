package apply

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	ErrOwnerNotFound = errors.New("owner not found")
)

func getGVK(gvkLabel string, gvk *schema.GroupVersionKind) error {
	parts := strings.Split(gvkLabel, ", Kind=")
	if len(parts) != 2 {
		return fmt.Errorf("invalid GVK format: %s", gvkLabel)
	}
	gvk.Group, gvk.Version, _ = strings.Cut(parts[0], "/")
	gvk.Kind = parts[1]
	return nil
}

func (a apply) FindOwner(ctx context.Context, obj kclient.Object) (kclient.Object, error) {
	if obj == nil {
		return nil, ErrOwnerNotFound
	}

	a.ctx = ctx

	var (
		gvkLabel  = obj.GetAnnotations()[LabelGVK]
		namespace = obj.GetAnnotations()[LabelNamespace]
		name      = obj.GetAnnotations()[LabelName]
		gvk       schema.GroupVersionKind
	)

	if gvkLabel == "" {
		return nil, ErrOwnerNotFound
	}

	if err := getGVK(gvkLabel, &gvk); err != nil {
		return nil, err
	}

	return a.get(gvk, nil, namespace, name)
}

func (a apply) PurgeOrphan(ctx context.Context, obj kclient.Object) error {
	if obj == nil {
		return nil
	}

	a.ctx = ctx

	if _, err := a.FindOwner(ctx, obj); apierrors.IsNotFound(err) {
		gvk, err := apiutil.GVKForObject(obj, a.client.Scheme())
		if err != nil {
			return err
		}

		return a.delete(gvk, obj.GetNamespace(), obj.GetName())
	} else if err == ErrOwnerNotFound {
		return nil
	} else if err != nil {
		return err
	}
	return nil
}
