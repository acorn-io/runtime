package cli

import (
	"context"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/imagesource"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
)

func getAuthForImage(ctx context.Context, clientFactory ClientFactory, image string) (*apiv1.RegistryAuth, error) {
	if tags.IsLocalReference(image) {
		return nil, nil
	}

	c, err := clientFactory.CreateDefault()
	if err != nil {
		return nil, err
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		// not failing on malformed image names
		return nil, nil
	}

	creds, err := imagesource.GetCreds(clientFactory.AcornConfigFile(), c)
	if err != nil {
		return nil, err
	}

	auth, _, err := creds(ctx, ref.Context().RegistryStr())
	return auth, err
}
