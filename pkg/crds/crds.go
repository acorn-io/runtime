package crds

import (
	"context"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/schemer/crd"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Create(ctx context.Context, scheme *runtime.Scheme, gvs ...schema.GroupVersion) error {
	var schemerCRDs []crd.CRD

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
				schemerCRDs = append(schemerCRDs, crd.CRD{
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

	client, err := k8sclient.New(restConfig)
	if err != nil {
		return err
	}

	factory, err := crd.NewFactoryFromClient(restConfig, scheme, func(objs ...runtime.Object) error {
		var kobjs []kclient.Object
		for _, obj := range objs {
			kobjs = append(kobjs, obj.(kclient.Object))
		}
		return apply.New(client).Ensure(ctx, kobjs...)
	})
	if err != nil {
		return err
	}

	if err := factory.BatchCreateCRDs(ctx, schemerCRDs...).BatchWait(); err != nil && !system.IsLocal() {
		return err
	}

	return nil
}
