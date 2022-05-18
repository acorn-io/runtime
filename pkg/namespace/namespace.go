package namespace

import (
	"context"
	"encoding/json"

	"github.com/acorn-io/acorn/pkg/labels"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ParentMost(ctx context.Context, c client.Client, nsName string) (*corev1.Namespace, error) {
	for {
		ns := &corev1.Namespace{}
		err := c.Get(ctx, client.ObjectKey{
			Name: nsName,
		}, ns)
		if err != nil {
			return nil, err
		}
		nsName = ns.Labels[labels.AcornAppNamespace]
		if nsName == "" {
			return ns, nil
		}
	}
}

func Selector(ctx context.Context, c client.Client, nsName string) (klabels.Selector, error) {
	sel := klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	})

	if nsName == "" {
		return sel, nil
	}

	children, err := Descendants(ctx, c, nsName)
	if err != nil {
		return nil, err
	}

	req, err := klabels.NewRequirement(labels.AcornAppNamespace, selection.In, children.List())
	if err != nil {
		return nil, err
	}

	return sel.Add(*req), nil
}

func Descendants(ctx context.Context, c client.Client, nsName string) (sets.String, error) {
	if nsName == "" {
		return nil, nil
	}

	ns := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{
		Name: nsName,
	}, ns)
	if err != nil {
		return nil, err
	}

	children, err := Children(ns)
	if err != nil {
		return nil, err
	}

	result := sets.NewString(nsName)
	result.Insert(maps.Values(children)...)

	return result, nil
}

func SetChildren(ns *corev1.Namespace, childrenMap map[string]string) error {
	childData, err := json.Marshal(childrenMap)
	if err != nil {
		return err
	}

	if ns.Annotations == nil {
		ns.Annotations = map[string]string{}
	}
	ns.Annotations[labels.AcornChildNamespaces] = string(childData)
	return nil
}

func Children(ns *corev1.Namespace) (map[string]string, error) {
	childData := map[string]string{}
	children := ns.Annotations[labels.AcornChildNamespaces]
	if len(children) > 0 {
		if err := json.Unmarshal([]byte(children), &childData); err != nil {
			return nil, err
		}
	}

	return childData, nil
}
