package credentials

import (
	"context"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"net/http"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	strategy.CompleteStrategy
	rest.TableConvertor
}

func NewStrategy(c kclient.WithWatch, expose bool) *Strategy {
	return &Strategy{
		CompleteStrategy: translation.NewTranslationStrategy(&Translator{
			client: c,
			expose: expose,
		}, remote.NewRemote(&corev1.Secret{}, &corev1.SecretList{}, c)),
		TableConvertor: tables.CredentialConverter,
	}
}

func (s *Strategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	s.PrepareForCreate(ctx, obj)
}

func (s *Strategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	cred := obj.(*apiv1.Credential)
	cred.ServerAddress = normalizeDockerIO(cred.ServerAddress)
}

// credentialValidate takes a username, password and serverAddress string to validate
// whether their combination is valid and will succeed login for pushes/pulls.
func (s *Strategy) CredentialValidate(ctx context.Context, username, password, serverAddress string) error {
	// Build a registry struct for the host
	reg, err := name.NewRegistry(serverAddress)
	if err != nil {
		return err
	}

	// Build a new transport for the registry which validates authentication
	scopes := []string{transport.PullScope, transport.PushScope}
	auth := &authn.Basic{Username: username, Password: password}
	_, err = transport.NewWithContext(ctx, reg, auth, http.DefaultTransport, scopes)
	if err != nil {
		return err
	}

	return nil
}
