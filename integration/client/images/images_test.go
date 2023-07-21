package images

import (
	"crypto"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	client2 "github.com/acorn-io/runtime/integration/client"
	"github.com/acorn-io/runtime/integration/helper"
	"github.com/acorn-io/runtime/pkg/client"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v2/pkg/signature"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	_ "embed"
)

func TestImageListGetDelete(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, project.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]

	assert.Equal(t, imageID, image.Name)
	assert.False(t, strings.HasPrefix(imageID, "sha256:"))
	assert.Equal(t, "sha256:"+image.Name, image.Digest)

	newImage, err := c.ImageGet(ctx, image.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, image.Name, newImage.Name)

	delImage, _, err := c.ImageDelete(ctx, image.Name, &client.ImageDeleteOptions{Force: false})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, newImage.Name, delImage.Name)

	_, err = c.ImageGet(ctx, image.Name)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestImageTagMove(t *testing.T) {
	c, project := helper.ClientAndProject(t)
	ctx := helper.GetCTX(t)

	imageID := client2.NewImage(t, project.Name)
	image2, err := c.AcornImageBuild(ctx, "../testdata/nginx2/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "../testdata/nginx2",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, imageID, "foo")
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, image2.ID, "foo:latest")
	if err != nil {
		t.Fatal(err)
	}

	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, image := range images {
		if image.Name == imageID {
			assert.Equal(t, []string([]string(nil)), image.Tags)
		} else if image.Name == image2.ID {
			assert.Equal(t, "foo:latest", image.Tags[0])
		} else {
			t.Fatal(err, "invalid image")
		}
	}
}

func TestImageTag(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	_ = client2.NewImage(t, project.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]

	err = c.ImageTag(ctx, image.Name, "foo:latest")
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, "foo:latest", "bar:latest")
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, "foo:latest", "ghcr.io/acorn-io/runtime/test:v0.0.0-abc")
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, "ghcr.io/acorn-io/runtime/test:v0.0.0-abc", "ghcr.io/acorn-io/runtime/test:v0.0.0-def")
	if err != nil {
		t.Fatal(err)
	}

	newImage, err := c.ImageGet(ctx, "ghcr.io/acorn-io/runtime/test:v0.0.0-abc")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, image.Name, newImage.Name)
	assert.Equal(t, "ghcr.io/acorn-io/runtime/test:v0.0.0-abc", newImage.Tags[2])
}

func TestImagePush(t *testing.T) {
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

	_ = client2.NewImage(t, project.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]
	tagName := registry + "/test:ci"

	err = c.ImageTag(ctx, image.Name, tagName)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}
}

func TestImagePull(t *testing.T) {
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
	tagName := registry + "/test:ci"

	err = c.ImageTag(ctx, id, tagName)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	project = helper.TempProject(t, kclient)

	c, err = client.New(restConfig, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	progress, err = c.ImagePull(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]
	assert.Equal(t, tagName, image.Tags[0])
}

func TestImageDetails(t *testing.T) {
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

	project = helper.TempProject(t, kclient)

	c, err = client.New(restConfig, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, project.Name)

	err = c.ImageTag(ctx, imageID, "foo")
	if err != nil {
		t.Fatal(err)
	}

	details, err := c.ImageDetails(ctx, imageID[:3], nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(details.AppImage.Acornfile, "nginx"))

	_, err = c.ImageDetails(ctx, "a12", nil)
	assert.True(t, apierrors.IsNotFound(err))

	details, err = c.ImageDetails(ctx, "foo:latest", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(details.AppImage.Acornfile, "nginx"))

	details, err = c.ImageDetails(ctx, remoteTagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(details.AppImage.Acornfile, "nginx"))

	// Test an auto-upgrade pattern that matches no local images, and make sure the proper error is returned
	_, err = c.ImageDetails(ctx, "dne:v#.#.#", nil)
	if err == nil {
		t.Fatal("expected error for auto-upgrade pattern that matches no local images")
	}
	assert.ErrorContains(t, err, "unable to find an image for dne:v#.#.# matching pattern v#.#.# - if you are trying to use a remote image, specify the full registry")
}

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

	assert.Empty(t, details.SignatureDigest, "signature digest should be empty")

	sigSigner, err := signature.SignerVerifierFromKeyRef(ctx, "./testdata/cosign.key", func(_ bool) ([]byte, error) { return []byte(""), nil })
	if err != nil {
		t.Fatal(err)
	}

	payload, sig, err := sigsig.SignImage(sigSigner, targetDigest, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	signatureB64 := base64.StdEncoding.EncodeToString(sig)

	imageSignOpts := &client.ImageSignOptions{}

	pubkey, err := sigSigner.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	pem, _, err := acornsign.PemEncodeCryptoPublicKey(pubkey)
	if err != nil {
		t.Fatal(err)
	}

	if pubkey != nil {
		imageSignOpts.PublicKey = string(pem)
	}

	nsig, err := c.ImageSign(ctx, targetDigest.String(), payload, signatureB64, imageSignOpts)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, nsig.SignatureDigest, "signature digest should not be empty")
	t.Logf("signature digest: %s", nsig.SignatureDigest)

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
	if err != nil {
		t.Fatal(err)
	}

	// 1.3 - Details with Signature Hash
	details, err = c.ImageDetails(ctx, targetDigest.DigestStr(), nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, details.SignatureDigest)

	// 2.1 - VERIFY invalid

	vOpts.Annotations = map[string]string{
		"foo": "bar",
	}

	_, err = c.ImageVerify(ctx, targetDigest.String(), vOpts)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImageDeleteTwoTags(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, project.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]

	assert.Equal(t, imageID, image.Name)
	assert.False(t, strings.HasPrefix(imageID, "sha256:"))
	assert.Equal(t, "sha256:"+image.Name, image.Digest)

	err = c.ImageTag(ctx, image.Name, "repo:tag1")
	if err != nil {
		t.Fatal(err)
	}
	err = c.ImageTag(ctx, image.Name, "repo:tag2")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = c.ImageDelete(ctx, image.Name, &client.ImageDeleteOptions{Force: false})
	expectedErr := fmt.Errorf("unable to delete %s (must be forced) - image is referenced in multiple repositories", image.Name)
	assert.Equal(t, expectedErr, err)

	_, _, err = c.ImageDelete(ctx, image.Name, &client.ImageDeleteOptions{Force: true})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.ImageGet(ctx, image.Name)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestImageBadTag(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, project.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]

	assert.Equal(t, imageID, image.Name)
	assert.False(t, strings.HasPrefix(imageID, "sha256:"))
	assert.Equal(t, "sha256:"+image.Name, image.Digest)

	err = c.ImageTag(ctx, image.Name, "foo:a@badtag")
	assert.Equal(t, "could not parse reference: foo:a@badtag", err.Error())

	err = c.ImageTag(ctx, image.Name, "foo@@:badtag")
	assert.Equal(t, "could not parse reference: foo@@:badtag", err.Error())
}

func TestImageCopy(t *testing.T) {
	helper.StartController(t)
	helper.StartAPI(t)
	registry, closeRegistry := helper.StartRegistry(t)
	t.Cleanup(closeRegistry)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)

	// Step 1: build an image and copy it to the registry
	image, err := c.AcornImageBuild(ctx, "../testdata/nginx/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "../testdata/nginx",
	})
	if err != nil {
		t.Fatal(err)
	}

	remoteTagName := registry + "/test:ci"

	progress, err := c.ImageCopy(ctx, image.ID, remoteTagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	// Step 2: build a second image and attempt to copy it to the registry - should fail because "force" is not set
	image2, err := c.AcornImageBuild(ctx, "../testdata/sidecar/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "../testdata/sidecar",
	})
	if err != nil {
		t.Fatal(err)
	}

	progress, err = c.ImageCopy(ctx, image2.ID, remoteTagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	var errorFound bool
	for update := range progress {
		if update.Error != "" {
			errorFound = true
			assert.Contains(t, update.Error, "not copying image")
			assert.Contains(t, update.Error, "since it already exists")
		}
	}
	assert.True(t, errorFound)

	// Now that it failed, force copy it to make sure that works
	progress, err = c.ImageCopy(ctx, image2.ID, remoteTagName, &client.ImageCopyOptions{Force: true})
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	// Step 4: copy the first image again, this time under a different tag
	progress, err = c.ImageCopy(ctx, image.ID, remoteTagName+"-2", nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	// Step 5: copy all tags from the first image in the registry to a new image in the registry
	oldRemoteRepo := registry + "/test"
	newRemoteRepo := registry + "/test2"
	progress, err = c.ImageCopy(ctx, oldRemoteRepo, newRemoteRepo, &client.ImageCopyOptions{AllTags: true})
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	// Step 6: verify that both tags exist in the registry
	repo, err := name.NewRepository(newRemoteRepo)
	if err != nil {
		t.Fatal(err)
	}

	tags, err := remote.List(repo, remote.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, tags, 2)
	assert.Contains(t, tags, "ci")
	assert.Contains(t, tags, "ci-2")
}
