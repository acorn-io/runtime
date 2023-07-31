package apps

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type nestedValidator struct{}

func (s nestedValidator) ValidateName(_ context.Context, obj runtime.Object) (result field.ErrorList) {
	name := obj.(kclient.Object).GetName()
	for _, piece := range strings.Split(name, ".") {
		if errs := validation.IsDNS1035Label(piece); len(errs) > 0 {
			result = append(result, field.Invalid(field.NewPath("metadata", "name"), name, strings.Join(errs, ",")))
		}
	}
	return
}
