package signatures

import (
	"testing"

	client2 "github.com/acorn-io/runtime/integration/client"
	"github.com/acorn-io/runtime/integration/helper"
	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/config"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	cclient "sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "embed"
)

//go:embed testdata/cosign.pub
var testPubKey []byte

func TestImageAllowRules(t *testing.T) {
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
	// enable image allow rules in acorn config

	cfg, err := config.Get(ctx, kclient)
	if err != nil {
		t.Fatal(err)
	}

	iarFeatureStateOriginal := cfg.Features[profiles.FeatureImageAllowRules]

	cfg.Features = map[string]bool{
		profiles.FeatureImageAllowRules: true,
	}

	err = config.Set(ctx, kclient, cfg)
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

	details, err := c.ImageDetails(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	ref, err := name.ParseReference(tagName)
	if err != nil {
		t.Fatal(err)
	}

	targetDigest := ref.Context().Digest(details.AppImage.Digest)

	assert.Empty(t, details.SignatureDigest, "signature digest should be empty before signing")

	// create image allow rule
	iar := &v1.ImageAllowRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: c.GetNamespace(),
		},
		Images: []string{"foobar"}, // does not cover the image
	}

	err = kclient.Create(ctx, iar)
	if err != nil {
		t.Fatal(err)
	}

	// try to run - expect failure
	_, err = c.AppRun(ctx, tagName, nil)
	assert.Error(t, err, "should error since image is not covered by images scope of IAR")

	// update image allow rule to cover that image
	iar.Images = []string{tagName}

	err = kclient.Update(ctx, iar)
	if err != nil {
		t.Fatal(err)
	}

	var niar v1.ImageAllowRule
	err = kclient.Get(ctx, cclient.ObjectKeyFromObject(iar), &niar)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, iar.Images, niar.Images)

	// try to run - expect success
	app, err := c.AppRun(ctx, tagName, nil)
	assert.NoError(t, err, "should not error since image is covered by images scope of IAR and there are not other rules")

	// remove app
	_, err = c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	// update iar to require a signature
	iar.Signatures = internalv1.ImageAllowRuleSignatures{
		Rules: []internalv1.SignatureRules{
			{
				SignedBy: internalv1.SignedBy{
					AllOf: []string{string(testPubKey)},
				},
			},
		},
	}

	err = kclient.Update(ctx, iar)
	if err != nil {
		t.Fatal(err)
	}

	// try to run - expect failure
	_, err = c.AppRun(ctx, tagName, nil)
	assert.Error(t, err, "should error since image is not signed by the required key")

	// sign image
	nsig, err := signImage(ctx, c, targetDigest, "./testdata/cosign.key")
	if err != nil {
		t.Fatal(err)
	}

	// push again (this should include the signature)
	progress, err = c.ImagePush(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	assert.NotEmpty(t, nsig.SignatureDigest, "signature should not be empty")

	ndetails, err := c.ImageDetails(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Image %s has signature %s", tagName, ndetails.SignatureDigest)
	assert.NotEmpty(t, ndetails.SignatureDigest, "signature digest should not be empty after signing")

	// try to run - expect success
	app, err = c.AppRun(ctx, tagName, nil)
	assert.NoError(t, err, "should not error since image is signed by the required key")

	// remove app
	_, err = c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	// update iar to require a signature with specific annotation
	iar.Signatures = internalv1.ImageAllowRuleSignatures{
		Rules: []internalv1.SignatureRules{
			{
				SignedBy: internalv1.SignedBy{
					AllOf: []string{string(testPubKey)},
				},
				Annotations: internalv1.SignatureAnnotations{
					Match: map[string]string{
						"foo": "bar",
					},
				},
			},
		},
	}

	err = kclient.Update(ctx, iar)
	if err != nil {
		t.Fatal(err)
	}

	// try to run - expect failure
	_, err = c.AppRun(ctx, tagName, nil)
	assert.Error(t, err, "should error since image is signed by the required key but does not match the required annotation")

	// Reset feature state to original value (especially heplful when testing locally)
	cfg.Features = map[string]bool{
		profiles.FeatureImageAllowRules: iarFeatureStateOriginal,
	}

	err = config.Set(ctx, kclient, cfg)
	if err != nil {
		t.Fatal(err)
	}
}
