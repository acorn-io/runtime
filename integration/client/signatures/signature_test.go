package signatures

import (
	"crypto"
	"testing"

	client2 "github.com/acorn-io/runtime/integration/client"
	"github.com/acorn-io/runtime/integration/helper"
	"github.com/acorn-io/runtime/pkg/client"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

func TestImageSignature(t *testing.T) {
	helper.StartController(t)
	registry, close := helper.StartRegistry(t)
	defer close()
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	id := client2.NewImage(t, project.Name)
	remoteTagName := registry + "/test:ci"

	err = c.ImageTag(ctx, id, remoteTagName)
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
	sig, err := signImage(ctx, c, targetDigest, "./testdata/cosign.key")
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

	_, err = c.ImageVerify(ctx, targetDigest.String(), vOpts)
	assert.NoError(t, err, "expected no error when verifying image with valid public key")

	// 1.3 - Details with Signature Hash
	details, err = c.ImageDetails(ctx, targetDigest.DigestStr(), nil)
	assert.NoError(t, err, "expected no error when getting details of signed image")

	assert.NotEmpty(t, details.SignatureDigest)

	// 2.1 - VERIFY invalid

	vOpts.Annotations = map[string]string{
		"foo": "bar",
	}

	_, err = c.ImageVerify(ctx, targetDigest.String(), vOpts)
	assert.Error(t, err, "expected error when verifying image with invalid annotations")
}
