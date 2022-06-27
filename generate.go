//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/internal.acorn.io/v1/
//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/api.acorn.io/v1/
//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/ui.acorn.io/v1/
//go:generate go run k8s.io/kube-openapi/cmd/openapi-gen -i github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1,github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/version,k8s.io/apimachinery/pkg/api/resource,k8s.io/api/core/v1,k8s.io/api/rbac/v1 -p ./pkg/openapi/generated -h tools/header.txt
//#go:generate go run k8s.io/code-generator/cmd/conversion-gen -i github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1 -p ./pkg/test/generated -h tools/header.txt

package main
