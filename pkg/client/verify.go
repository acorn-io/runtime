package client

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

func (c *DefaultClient) ImageVerify(ctx context.Context, image string, opts *ImageVerifyOptions) (*apiv1.ImageSignature, error) {
	sigInput := &apiv1.ImageSignature{
		PublicKey:    opts.PublicKey,
		Auth:         opts.Auth,
		NoVerifyName: opts.NoVerifyName,
	}

	if opts.PublicKey == "" {
		return nil, fmt.Errorf("public key required for verification")
	}

	sigInput.Annotations = internalv1.SignatureAnnotations{
		Match: opts.Annotations,
	}

	sigResult := &apiv1.ImageSignature{}
	err := c.RESTClient.Post().
		Namespace(c.Namespace).
		Resource("images").
		Name(strings.ReplaceAll(image, "/", "+")).
		SubResource("verify").
		Body(sigInput).Do(ctx).Into(sigResult)

	return sigResult, err
}
