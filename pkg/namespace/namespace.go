package namespace

import (
	"context"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/endpoints/request"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DenormalizeName(ctx context.Context, c client.Client, namespace, name string) (string, string, error) {
	ns, _ := request.NamespaceFrom(ctx)
	for {
		prefix, suffix, ok := strings.Cut(name, ".")
		if !ok {
			break
		}
		app := &v1.AppInstance{}
		err := c.Get(ctx, router.Key(ns, prefix), app)
		if err != nil {
			return ns, name, err
		}

		name = suffix
		ns = app.Status.Namespace
	}

	return ns, name, nil
}

func NormalizedName(obj metav1.ObjectMeta) (string, string) {
	ns := obj.Namespace
	name := obj.Name

	rootNS := obj.Labels[labels.AcornAppNamespace]
	if rootNS != "" {
		ns = rootNS
	}
	if len(obj.Labels[labels.AcornAppName]) > 0 {
		name = obj.Labels[labels.AcornAppName] + "." + obj.Name
	}
	return ns, name
}

func Selector(ctx context.Context) klabels.Selector {
	sel := klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	})

	nsName, _ := request.NamespaceFrom(ctx)
	if nsName == "" {
		return sel
	}

	return klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged:      "true",
		labels.AcornAppNamespace: nsName,
	})
}
