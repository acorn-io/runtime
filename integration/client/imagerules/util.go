package imagerules

import (
	"context"
	"encoding/base64"

	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/v2/pkg/signature"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
)

func signImage(ctx context.Context, c client.Client, targetDigest name.Digest, targetName, key string) (*v1.ImageSignature, error) {
	sigSigner, err := signature.SignerVerifierFromKeyRef(ctx, key, func(_ bool) ([]byte, error) { return []byte(""), nil })
	if err != nil {
		return nil, err
	}

	annotations := map[string]string{
		acornsign.SignatureAnnotationSignedName: targetName,
	}

	iannotations := map[string]interface{}{}
	for k, v := range annotations {
		iannotations[k] = v
	}

	payload, sig, err := sigsig.SignImage(sigSigner, targetDigest, iannotations)
	if err != nil {
		return nil, err
	}

	signatureB64 := base64.StdEncoding.EncodeToString(sig)

	imageSignOpts := &client.ImageSignOptions{}

	pubkey, err := sigSigner.PublicKey()
	if err != nil {
		return nil, err
	}

	pem, _, err := acornsign.PemEncodeCryptoPublicKey(pubkey)
	if err != nil {
		return nil, err
	}

	if pubkey != nil {
		imageSignOpts.PublicKey = string(pem)
	}

	nsig, err := c.ImageSign(ctx, targetDigest.String(), payload, signatureB64, imageSignOpts)
	if err != nil {
		return nil, err
	}

	return nsig, nil
}
