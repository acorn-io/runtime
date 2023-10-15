package secrets

import (
	"context"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	sec "github.com/acorn-io/runtime/pkg/secrets"
)

type defaultSecretGenerateStrategy struct {
	strategy strategy.CompleteStrategy
}

func (d *defaultSecretGenerateStrategy) Create(ctx context.Context, object types.Object) (types.Object, error) {
	secret := object.(*apiv1.Secret)
	// If the secret is of type 'basic' and data is empty,
	// default username and password values are set.
	if secret.Type == "basic" && secret.Data == nil {
		username, err := sec.GenerateRandomSecret(8, "")
		if err != nil {
			return nil, err
		}

		password, err := sec.GenerateRandomSecret(16, "")
		if err != nil {
			return nil, err
		}

		secret.Data = map[string][]byte{}

		secret.Data["username"] = []byte(username)
		secret.Data["password"] = []byte(password)
	}
	return d.strategy.Create(ctx, secret)
}

func (d *defaultSecretGenerateStrategy) New() types.Object {
	return &apiv1.Secret{}
}
