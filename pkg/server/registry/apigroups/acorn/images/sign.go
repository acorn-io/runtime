package images

import (
	"context"
	"crypto"
	"fmt"
	"net/http"
	"strings"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/acorn-io/mink/pkg/validator"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/imagedetails"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	cremote "github.com/sigstore/cosign/v2/pkg/cosign/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
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
	opts := []remote.Option{t.transportOpt}
	if isig.Auth != nil {
		imageName := strings.ReplaceAll(isig.Name, "+", "/")
		ref, err := name.ParseReference(imageName)
		if err == nil {
			opts = append(opts, remote.WithAuthFromKeychain(images.NewSimpleKeychain(ref.Context(), *isig.Auth, nil)))
		}
	}

	isig.Name = strings.ReplaceAll(isig.Name, "+", "/")

	isig.SignatureDigest, err = t.ImageSign(ctx, ns, *isig, opts...)
	if err != nil {
		return nil, err
	}

	return isig, nil
}

func (t *ImageSignStrategy) New() types.Object {
	return &apiv1.ImageSignature{}
}

func (t *ImageSignStrategy) ImageSign(ctx context.Context, namespace string, signature apiv1.ImageSignature, remoteOpts ...remote.Option) (string, error) {
	imageDetails, err := imagedetails.GetImageDetails(ctx, t.client, namespace, signature.Name, nil, nil, "", false, remoteOpts...)
	if err != nil {
		return "", err
	}

	targetName := imageDetails.Name

	var mutateOpts []mutate.SignOption

	if signature.PublicKey != nil {
		verifier, err := acornsign.DecodePEM(signature.PublicKey, crypto.SHA256)
		if err != nil {
			return "", err
		}
		dupeDetector := cremote.NewDupeDetector(verifier)

		mutateOpts = append(mutateOpts, mutate.WithDupeDetector(dupeDetector))
	}

	ref, err := images.GetImageReference(ctx, t.client, namespace, targetName)
	if err != nil {
		return "", err
	}

	ociEntity, err := ociremote.SignedEntity(ref, ociremote.WithRemoteOptions(remoteOpts...))
	if err != nil {
		return "", fmt.Errorf("accessing entity: %w", err)
	}

	signatureOCI, err := static.NewSignature(signature.Payload, signature.SignatureB64)
	if err != nil {
		return "", err
	}

	sigOCIDigest, err := signatureOCI.Digest()
	if err != nil {
		return "", err
	}

	mutatedOCIEntity, err := mutate.AttachSignatureToEntity(ociEntity, signatureOCI, mutateOpts...)
	if err != nil {
		return "", err
	}

	targetRepo := ref.Context()
	if err := ociremote.WriteSignatures(targetRepo, mutatedOCIEntity, ociremote.WithRemoteOptions(remoteOpts...)); err != nil {
		return "", err
	}

	return sigOCIDigest.String(), nil
}
