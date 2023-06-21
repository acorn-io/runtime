package openapi

import (
	"github.com/acorn-io/runtime/pkg/openapi/generated"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	result := generated.GetOpenAPIDefinitions(ref)
	for _, v := range result {
		for name := range v.Schema.SchemaProps.Properties {
			if name == "deployArgs" || name == "buildArgs" || name == "params" {
				v.Schema.SchemaProps.Properties[name] = spec.Schema{
					VendorExtensible: spec.VendorExtensible{
						Extensions: spec.Extensions{
							"x-kubernetes-preserve-unknown-fields": "true",
						},
					},
				}
			}
		}
	}
	return result
}
