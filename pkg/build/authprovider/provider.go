package authprovider

import (
	"context"
	"fmt"

	"github.com/docker/docker-credential-helpers/registryurl"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewProvider(keychain authn.Keychain) session.Attachable {
	return &AuthProvider{keychain: keychain}
}

type AuthProvider struct {
	keychain authn.Keychain
}

func (a *AuthProvider) Register(server *grpc.Server) {
	auth.RegisterAuthServer(server, a)
}

func (a *AuthProvider) Credentials(_ context.Context, request *auth.CredentialsRequest) (*auth.CredentialsResponse, error) {
	if a.keychain == nil {
		return &auth.CredentialsResponse{}, nil
	}
	u, err := registryurl.Parse(request.Host)
	if err != nil {
		return nil, err
	}
	reg, err := name.NewRegistry(u.Host)
	if err != nil {
		return nil, err
	}
	authenticator, err := a.keychain.Resolve(reg)
	if err != nil {
		return nil, err
	}
	resolved, err := authenticator.Authorization()
	if err != nil {
		return nil, err
	}
	return &auth.CredentialsResponse{
		Username: resolved.Username,
		Secret:   resolved.Password,
	}, nil
}

func (a *AuthProvider) FetchToken(context.Context, *auth.FetchTokenRequest) (*auth.FetchTokenResponse, error) {
	return nil, fmt.Errorf("not supported")
}

func (a *AuthProvider) GetTokenAuthority(context.Context, *auth.GetTokenAuthorityRequest) (*auth.GetTokenAuthorityResponse, error) {
	return nil, status.Errorf(codes.Unavailable, "client side tokens disabled")
}

func (a *AuthProvider) VerifyTokenAuthority(context.Context, *auth.VerifyTokenAuthorityRequest) (*auth.VerifyTokenAuthorityResponse, error) {
	return nil, fmt.Errorf("not supported")
}
