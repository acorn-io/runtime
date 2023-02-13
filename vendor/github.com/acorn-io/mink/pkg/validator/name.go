package validator

import (
	"context"
	"strings"

	"github.com/acorn-io/mink/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var ValidDNSLabel = &validDNSLabel{}

type validDNSLabel struct {
}

func (v *validDNSLabel) ValidateName(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	name := obj.(types.Object).GetName()
	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), name, strings.Join(errs, ",")))
	}
	return
}

var ValidDNSSubdomain = &validDNSSubdomain{}

type validDNSSubdomain struct {
}

func (v *validDNSSubdomain) ValidateName(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	name := obj.(types.Object).GetName()
	if errs := validation.IsDNS1123Subdomain(name); len(errs) > 0 {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), name, strings.Join(errs, ",")))
	}
	return
}

var NoValidation = &noValidation{}

type noValidation struct {
}

func (n *noValidation) ValidateName(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	return nil
}
