package imageallowrules

import (
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	imagename "github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SimpleImageScope string

const (
	SimpleImageScopeRegistry   SimpleImageScope = "registry"
	SimpleImageScopeRepository SimpleImageScope = "repository"
	SimpleImageScopeExact      SimpleImageScope = "exact"
	SimpleImageScopeAll        SimpleImageScope = "all"
)

func GenerateSimpleAllowRule(namespace string, name string, image string, scope string) (*v1.ImageAllowRule, error) {
	ref, err := imagename.ParseReference(image, imagename.WithDefaultTag(""), imagename.WithDefaultRegistry(""))
	if err != nil {
		return nil, fmt.Errorf("error parsing image: %w", err)
	}

	is, err := buildImageScope(ref, scope)
	if err != nil {
		return nil, err
	}

	return &v1.ImageAllowRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Images: []string{is},
	}, nil
}

func buildImageScope(image imagename.Reference, scope string) (string, error) {
	var is string

	switch SimpleImageScope(scope) {
	case SimpleImageScopeRegistry:
		is = fmt.Sprintf("%s/**", image.Context().RegistryStr())
	case SimpleImageScopeRepository:
		is = fmt.Sprintf("%s/%s:**", image.Context().RegistryStr(), image.Context().RepositoryStr())
	case SimpleImageScopeExact:
		is = image.Name()
	case SimpleImageScopeAll:
		is = "**"
	default:
		return "", fmt.Errorf("invalid scope: %s", scope)
	}

	return is, nil
}
