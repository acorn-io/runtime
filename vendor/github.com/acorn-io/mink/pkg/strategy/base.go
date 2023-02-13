package strategy

import "k8s.io/apiserver/pkg/registry/rest"

type Base interface {
	rest.Storage
	rest.Scoper
	rest.TableConvertor
}
