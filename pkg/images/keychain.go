package images

import (
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/google/go-containerregistry/pkg/authn"
)

type SimpleKeychain struct {
	resource authn.Resource
	auth     apiv1.RegistryAuth
	next     authn.Keychain
}

func NewSimpleKeychain(resource authn.Resource, auth apiv1.RegistryAuth, next authn.Keychain) *SimpleKeychain {
	return &SimpleKeychain{
		resource: resource,
		auth:     auth,
		next:     next,
	}
}

func (s *SimpleKeychain) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{
		Username: s.auth.Username,
		Password: s.auth.Password,
	}, nil
}

func (s *SimpleKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	if resource.RegistryStr() == s.resource.RegistryStr() {
		return s, nil
	}
	if s.next == nil {
		return authn.Anonymous, nil
	}
	return s.next.Resolve(resource)
}
