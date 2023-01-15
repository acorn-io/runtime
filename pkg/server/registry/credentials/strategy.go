package credentials

import (
	"context"
	"net/http"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Strategy struct {
}

func (s *Strategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	s.PrepareForCreate(ctx, obj)
}

func (s *Strategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	cred := obj.(*apiv1.Credential)
	cred.ServerAddress = normalizeDockerIO(cred.ServerAddress)
}

func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.Credential)
	if !params.SkipChecks {
		if err := CredentialValidate(ctx, params.Username, *params.Password, params.ServerAddress); err != nil {
			result = append(result, field.Forbidden(field.NewPath("username/password"), err.Error()))
		}
	}
	return result
}
func (s *Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.Credential)
	return s.Validate(ctx, params)
}

// CredentialValidate takes a username, password and serverAddress string to validate
// whether their combination is valid and will succeed login for pushes/pulls.
func CredentialValidate(ctx context.Context, username, password, serverAddress string) error {
	// Build a registry struct for the host
	reg, err := name.NewRegistry(serverAddress)
	if err != nil {
		return err
	}

	// Build a new transport for the registry which validates authentication
	auth := &authn.Basic{Username: username, Password: password}
	_, err = transport.NewWithContext(ctx, reg, auth, http.DefaultTransport, nil)
	if err != nil {
		return err
	}

	return nil
}
