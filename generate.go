//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/internal.acorn.io/v1/
//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/api.acorn.io/v1/
//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/internal.admin.acorn.io/v1/
//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/admin.acorn.io/v1/
//go:generate go run k8s.io/kube-openapi/cmd/openapi-gen -i github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1,github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1,github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1,github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/version,k8s.io/apimachinery/pkg/api/resource,k8s.io/api/core/v1,k8s.io/api/rbac/v1,k8s.io/apimachinery/pkg/util/intstr,github.com/acorn-io/aml/pkg/jsonschema -p ./pkg/openapi/generated -h tools/header.txt
//go:generate go run github.com/golang/mock/mockgen --build_flags=--mod=mod -destination=./pkg/mocks/mock_client.go -package=mocks github.com/acorn-io/runtime/pkg/client Client,ProjectClientFactory
//go:generate go run github.com/golang/mock/mockgen --build_flags=--mod=mod -destination=./pkg/mocks/dns/mock.go -package=mocks github.com/acorn-io/runtime/pkg/dns Client
//go:generate go run github.com/golang/mock/mockgen --build_flags=--mod=mod -destination=./pkg/mocks/k8s/mock.go -package=mocks sigs.k8s.io/controller-runtime/pkg/client Reader

package main
