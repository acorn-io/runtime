package imagerules

import (
	"crypto"
	"testing"

	client2 "github.com/acorn-io/runtime/integration/client"
	"github.com/acorn-io/runtime/integration/helper"
	"github.com/acorn-io/runtime/pkg/client"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageSignVerify(t *testing.T) {
	helper.StartController(t)
	registry, cancel := helper.StartRegistry(t)
	defer cancel()

	ctx := helper.GetCTX(t)
	c, project := helper.ClientAndProject(t)

	id := client2.NewImage(t, project.Name)
	remoteTagName := registry + "/test:ci"

	err := c.ImageTag(ctx, id, remoteTagName)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, remoteTagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	details, err := c.ImageDetails(ctx, remoteTagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 1.1 - SIGN valid

	ref, err := name.ParseReference(remoteTagName)
	if err != nil {
		t.Fatal(err)
	}

	targetDigest := ref.Context().Digest(details.AppImage.Digest)

	assert.Empty(t, details.SignatureDigest, "signature digest should be empty before signing")

	// sign
	sig, err := signImage(ctx, c, targetDigest, remoteTagName, "./testdata/cosign.key")
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, sig.SignatureDigest, "signature digest should not be empty after signing")

	// 1.2 - VERIFY valid
	v, err := signature.VerifierForKeyRef(ctx, "./testdata/cosign.pub", crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}

	pubkey2, err := v.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	pem2, _, err := acornsign.PemEncodeCryptoPublicKey(pubkey2)
	if err != nil {
		t.Fatal(err)
	}
	vOpts := &client.ImageVerifyOptions{
		PublicKey: string(pem2),
	}

	_, err = c.ImageVerify(ctx, remoteTagName, vOpts)
	require.NoError(t, err, "expected no error when verifying image %s with valid public key", remoteTagName)

	_, err = c.ImageVerify(ctx, targetDigest.String(), vOpts)
	require.Error(t, err, "expected error when verifying image with valid public key but referenced by digest which was not used for signing (acorn.io/signedName annotation)")

	vOpts.NoVerifyName = true
	_, err = c.ImageVerify(ctx, targetDigest.String(), vOpts)
	require.NoError(t, err, "expected no error when verifying image with valid public key and digest as reference - where NoVerifyName is set")
	vOpts.NoVerifyName = false

	// Now sign again using the digest as the reference
	_, err = signImage(ctx, c, targetDigest, targetDigest.String(), "./testdata/cosign.key")
	require.NoError(t, err, "expected no error when signing image with digest as reference")

	// Verify again - this time successfully using the digest as the reference
	_, err = c.ImageVerify(ctx, targetDigest.String(), vOpts)
	require.NoError(t, err, "expected no error when verifying image with valid public key and digest as reference")

	// 1.3 - Details with Signature Hash
	details, err = c.ImageDetails(ctx, targetDigest.DigestStr(), nil)
	require.NoError(t, err, "expected no error when getting details of signed image")

	assert.NotEmpty(t, details.SignatureDigest)

	// 2.1 - VERIFY invalid

	vOpts.Annotations = map[string]string{
		"foo": "bar",
	}

	_, err = c.ImageVerify(ctx, targetDigest.String(), vOpts)
	require.Error(t, err, "expected error when verifying image with invalid annotations")
}
