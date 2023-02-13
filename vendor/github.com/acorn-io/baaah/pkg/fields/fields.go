package fields

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Fields interface {
	fields.Fields
	FieldNames() []string
}

func AddFieldConversion(scheme *runtime.Scheme, obj runtime.Object) error {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		// ignore errors in determining GVK
		return nil
	}
	f, ok := obj.(Fields)
	if !ok {
		return nil
	}
	return scheme.AddFieldLabelConversionFunc(gvk, ValidSelectors(f.FieldNames()...))
}

func AddKnownTypesWithFieldConversion(scheme *runtime.Scheme, gv schema.GroupVersion, types ...runtime.Object) error {
	for _, obj := range types {
		scheme.AddKnownTypes(gv, obj)
		if err := AddFieldConversion(scheme, obj); err != nil {
			return err
		}
	}
	return nil
}

func ValidSelectors(labels ...string) func(string, string) (string, string, error) {
	return func(label string, value string) (string, string, error) {
		for _, checkLabel := range labels {
			if label == checkLabel {
				return label, value, nil
			}
		}
		return runtime.DefaultMetaV1FieldSelectorConversion(label, value)
	}
}
