package k8sclient

import (
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/rancher/wrangler/pkg/name"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type FastMapper struct {
	mapper         meta.RESTMapper
	kindToResource map[string]string
	resourceToKind map[string]string
	clusterScoped  map[string]bool
}

func NewMapper(scheme *runtime.Scheme, mapper meta.RESTMapper) (meta.RESTMapper, error) {
	kindToResource := map[string]string{}
	resourceToKind := map[string]string{}
	clusterScoped := map[string]bool{}

	gv := schema.GroupVersion{Group: api.Group, Version: apiv1.Version}
	for kind := range scheme.KnownTypes(gv) {
		resource := name.GuessPluralName(strings.ToLower(kind))
		kindToResource[kind] = resource
		resourceToKind[resource] = kind
		obj, err := scheme.New(gv.WithKind(kind))
		if err != nil {
			return nil, err
		}
		if scoped, ok := obj.(strategy.NamespaceScoper); ok {
			if !scoped.NamespaceScoped() {
				clusterScoped[kind] = true
			}
		}
	}

	return &FastMapper{
		mapper:         mapper,
		kindToResource: kindToResource,
		resourceToKind: resourceToKind,
		clusterScoped:  clusterScoped,
	}, nil
}

func (f *FastMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	if resource.Group != api.Group {
		return f.mapper.KindFor(resource)
	}
	return schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    f.resourceToKind[resource.Resource],
	}, nil
}

func (f *FastMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	if resource.Group != api.Group {
		return f.mapper.KindsFor(resource)
	}
	gvk, err := f.KindFor(resource)
	return []schema.GroupVersionKind{gvk}, err
}

func (f *FastMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	if input.Group != api.Group {
		return f.mapper.ResourceFor(input)
	}
	return input, nil
}

func (f *FastMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	if input.Group != api.Group {
		return f.mapper.ResourcesFor(input)
	}
	ret, err := f.ResourceFor(input)
	return []schema.GroupVersionResource{ret}, err
}

func (f *FastMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	if gk.Group != api.Group {
		return f.mapper.RESTMapping(gk, versions...)
	}
	return &meta.RESTMapping{
		Resource: schema.GroupVersionResource{
			Group:    gk.Group,
			Version:  versions[0],
			Resource: f.kindToResource[gk.Kind],
		},
		GroupVersionKind: schema.GroupVersionKind{},
		Scope:            f.getScope(gk.Kind),
	}, nil
}

func (f *FastMapper) getScope(kind string) meta.RESTScope {
	if f.clusterScoped[kind] {
		return meta.RESTScopeRoot
	}
	return meta.RESTScopeNamespace
}

func (f *FastMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	if gk.Group != api.Group {
		return f.mapper.RESTMappings(gk, versions...)
	}
	ret, err := f.RESTMapping(gk, versions...)
	return []*meta.RESTMapping{ret}, err
}

func (f *FastMapper) ResourceSingularizer(resource string) (singular string, err error) {
	k, ok := f.resourceToKind[resource]
	if ok {
		return strings.ToLower(k), nil
	}
	return f.mapper.ResourceSingularizer(resource)
}
