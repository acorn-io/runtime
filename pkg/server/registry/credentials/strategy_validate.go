package credentials

import (
	"context"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"net/http"
)

// credentialValidate takes a username, password and serverAddress string to validate
// whether their combination is valid and will succeed login for pushes/pulls.
func (s *Strategy) credentialValidate(ctx context.Context, username, password, serverAddress string) error {
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
