package events

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// TODO(njhale): Validate things like TTL, etc

type validator struct{}

func newValidator() *validator {
	return &validator{}
}

func (s *validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	// TODO(njhale): Implement me!
	return result
}
