//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/internal.acorn.io/v1/
//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/api.acorn.io/v1/
//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/internal.admin.acorn.io/v1/
//go:generate go run github.com/acorn-io/baaah/cmd/deepcopy ./pkg/apis/admin.acorn.io/v1/
//go:generate go run k8s.io/kube-openapi/cmd/openapi-gen -i github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1,github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1,github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1,github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/version,k8s.io/apimachinery/pkg/api/resource,k8s.io/api/core/v1,k8s.io/api/rbac/v1 -p ./pkg/openapi/generated -h tools/header.txt
//go:generate $GOPATH/bin/mockgen --build_flags=--mod=mod -destination=./pkg/mocks/mock_client.go -package=mocks github.com/acorn-io/acorn/pkg/client Client,ProjectClientFactory

package main
