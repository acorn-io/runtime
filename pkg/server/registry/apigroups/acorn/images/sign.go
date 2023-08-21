package images

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/acorn-io/mink/pkg/validator"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	cremote "github.com/sigstore/cosign/v2/pkg/cosign/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewImageSign(c client.WithWatch, transport http.RoundTripper) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.ImageSignature{}).
		WithValidateName(validator.NoValidation).
		WithCreate(&ImageSignStrategy{
			client:       c,
			transportOpt: remote.WithTransport(transport),
		}).Build()
}

type ImageSignStrategy struct {
	client       client.WithWatch
	transportOpt remote.Option
}

func (t *ImageSignStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	var (
		isig = obj.(*apiv1.ImageSignature)
		err  error
	)

	if isig.Name == "" {
		ri, ok := request.RequestInfoFrom(ctx)
		if ok {
			isig.Name = ri.Name
		}
	}
	ns, _ := request.NamespaceFrom(ctx)

	isig.Name = strings.ReplaceAll(isig.Name, "+", "/")

	isig.SignatureDigest, err = t.ImageSign(ctx, ns, *isig)
	if err != nil {
		return nil, err
	}

	return isig, nil
}

func (t *ImageSignStrategy) New() types.Object {
	return &apiv1.ImageSignature{}
}

func (t *ImageSignStrategy) ImageSign(ctx context.Context, namespace string, signature apiv1.ImageSignature) (string, error) {
	ref, err := images.GetImageReference(ctx, t.client, namespace, signature.Name)
	if err != nil {
		return "", err
	}

	remoteOpts, err := images.GetAuthenticationRemoteOptionsWithLocalAuth(ctx, ref.Context(), signature.Auth, t.client, namespace, t.transportOpt)
	if err != nil {
		return "", err
	}

	var mutateOpts []mutate.SignOption

	if signature.PublicKey != "" {
		verifiers, err := acornsign.LoadVerifiers(ctx, signature.PublicKey, "sha256")
		if err != nil {
			return "", err
		}
		if len(verifiers) != 1 {
			return "", fmt.Errorf("expected exactly one verifier from public key %s, got %d", signature.PublicKey, len(verifiers))
		}

		dupeDetector := cremote.NewDupeDetector(verifiers[0])

		mutateOpts = append(mutateOpts, mutate.WithDupeDetector(dupeDetector))
	}

	targetEntity, err := ociremote.SignedEntity(ref, ociremote.WithRemoteOptions(remoteOpts...))
	if err != nil {
		return "", fmt.Errorf("accessing entity: %w", err)
	}

	signatureOCI, err := static.NewSignature(signature.Payload, signature.SignatureB64)
	if err != nil {
		return "", err
	}

	signedEntity, err := mutate.AttachSignatureToEntity(targetEntity, signatureOCI, mutateOpts...)
	if err != nil {
		return "", err
	}

	targetRepo := ref.Context()

	if err := ociremote.WriteSignatures(targetRepo, signedEntity, ociremote.WithRemoteOptions(remoteOpts...)); err != nil {
		return "", err
	}

	// Get the digest of the signature artifact we just wrote
	se, err := signedEntity.Signatures()
	if err != nil {
		return "", err
	}
	sigDigest, err := se.Digest()
	if err != nil {
		return "", err
	}
	logrus.Infof("Wrote signatures artifact %s to %s", sigDigest, targetRepo.Name())

	return sigDigest.String(), nil
}
