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
	"github.com/acorn-io/runtime/pkg/imagedetails"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewImageVerify(c client.WithWatch, transport http.RoundTripper) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.ImageSignature{}).
		WithValidateName(validator.NoValidation).
		WithCreate(&ImageVerifyStrategy{
			client:       c,
			transportOpt: remote.WithTransport(transport),
		}).Build()
}

type ImageVerifyStrategy struct {
	client       client.WithWatch
	transportOpt remote.Option
}

func (t *ImageVerifyStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	var (
		isig = obj.(*apiv1.ImageSignature)
	)

	if isig.PublicKey == "" {
		return nil, fmt.Errorf("public key is required for verification")
	}

	if isig.Name == "" {
		ri, ok := request.RequestInfoFrom(ctx)
		if ok {
			isig.Name = ri.Name
		}
	}
	ns, _ := request.NamespaceFrom(ctx)

	isig.Name = strings.ReplaceAll(isig.Name, "+", "/")

	return isig, t.ImageVerify(ctx, ns, *isig)
}

func (t *ImageVerifyStrategy) New() types.Object {
	return &apiv1.ImageSignature{}
}

func (t *ImageVerifyStrategy) ImageVerify(ctx context.Context, namespace string, signature apiv1.ImageSignature) error {
	ref, err := images.GetImageReference(ctx, t.client, namespace, signature.Name)
	if err != nil {
		return err
	}

	remoteOpts, err := images.GetAuthenticationRemoteOptionsWithLocalAuth(ctx, ref.Context(), signature.Auth, t.client, namespace, t.transportOpt)
	if err != nil {
		return err
	}

	// imageDetails to get image and signature digests
	imageDetails, err := imagedetails.GetImageDetails(ctx, t.client, namespace, signature.Name, nil, nil, "", false, remoteOpts...)
	if err != nil {
		return err
	}

	targetName := imageDetails.Name

	if imageDetails.SignatureDigest == "" {
		cerr := cosign.NewVerificationError(cosign.ErrNoSignaturesFoundMessage)
		cerr.(*cosign.VerificationError).SetErrorType(cosign.ErrNoSignaturesFoundType)
		return fmt.Errorf("%w: %s", cerr, targetName)
	}

	verifyOpts := &acornsign.VerifyOpts{
		AnnotationRules:    signature.Annotations,
		SignatureAlgorithm: "sha256",
		Key:                signature.PublicKey,
		NoCache:            false,
		ImageRef:           ref.Context().Digest(imageDetails.AppImage.Digest),
		SignatureRef:       ref.Context().Digest(imageDetails.SignatureDigest),
	}

	if err := verifyOpts.WithRemoteOpts(ctx, t.client, namespace, remoteOpts...); err != nil {
		return err
	}

	return acornsign.VerifySignature(ctx, *verifyOpts)
}
