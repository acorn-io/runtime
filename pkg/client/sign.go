package client

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
)

func (c *DefaultClient) ImageSign(ctx context.Context, image string, payload []byte, signatureB64 string, opts *ImageSignOptions) (*apiv1.ImageSignature, error) {
	sigInput := &apiv1.ImageSignature{
		Payload:      payload,
		SignatureB64: signatureB64,
	}

	if opts.PublicKey != nil {
		pem, _, err := acornsign.PemEncodeCryptoPublicKey(*opts.PublicKey)
		if err != nil {
			return nil, err
		}
		sigInput.PublicKey = pem
	}

	imageDetails, err := c.ImageDetails(ctx, image, &ImageDetailsOptions{})
	if err != nil {
		return nil, err
	}

	image = strings.ReplaceAll(imageDetails.AppImage.ID, "/", "+")

	sigResult := &apiv1.ImageSignature{}
	err = c.RESTClient.Post().
		Namespace(c.Namespace).
		Resource("images").
		Name(image).
		SubResource("sign").
		Body(sigInput).Do(ctx).Into(sigResult)

	return sigResult, err
}
