package events

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type validator struct{}

func (s validator) ValidateName(_ context.Context, obj runtime.Object) (result field.ErrorList) {
	e := obj.(*apiv1.Event)
	if e.Name != "" {
		result = append(result, field.Forbidden(
			field.NewPath("metadata", "name"),
			"can't be set explicitly, use metadata.generateName",
		))
	}

	return
}
