package crds

import (
	"context"

	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/rancher/wrangler/pkg/crd"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Create(ctx context.Context, scheme *runtime.Scheme, gvs ...schema.GroupVersion) error {
	var wranglerCRDs []crd.CRD

	for _, gv := range gvs {
		for kind := range scheme.KnownTypes(gv) {
			gvk := gv.WithKind(kind)
			obj, err := scheme.New(gvk)
			if err != nil {
				return err
			}
			_, isObj := obj.(kclient.Object)
			_, isListObj := obj.(kclient.ObjectList)
			if isObj && !isListObj {
				var nonNamespaced bool
				if o, ok := obj.(strategy.NamespaceScoper); ok {
					nonNamespaced = !o.NamespaceScoped()
				}
				wranglerCRDs = append(wranglerCRDs, crd.CRD{
					GVK:          gvk,
					SchemaObject: obj,
					Status:       true,
					NonNamespace: nonNamespaced,
				}.WithColumnsFromStruct(obj))
			}
		}
	}

	restConfig, err := restconfig.New(scheme)
	if err != nil {
		return err
	}

	factory, err := crd.NewFactoryFromClient(restConfig)
	if err != nil {
		return err
	}

	return factory.BatchCreateCRDs(ctx, wranglerCRDs...).BatchWait()
}
