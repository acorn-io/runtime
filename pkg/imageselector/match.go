package imageselector

import (
	"context"
	"fmt"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/images"
	nameselector "github.com/acorn-io/runtime/pkg/imageselector/name"
	signatureselector "github.com/acorn-io/runtime/pkg/imageselector/signatures"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ImageSelectorNoMatchError struct {
	ImageName string
	Field     string
	Err       error
}

type MatchImageOpts struct {
	SignatureOpts signatureselector.MatchImageSignatureOpts
}

func (e *ImageSelectorNoMatchError) Error() string {
	return fmt.Sprintf("image [%s] does not match selector field [%s]: %v", e.ImageName, e.Field, e.Err)
}

func MatchImage(ctx context.Context, c client.Reader, namespace, imageName, resolvedName, digest string, selector internalv1.ImageSelector, opts MatchImageOpts, remoteOpts ...remote.Option) error {
	imageNameRef, err := images.GetImageReference(ctx, c, namespace, imageName)
	if err != nil {
		return fmt.Errorf("error parsing image reference %s: %w", imageName, err)
	}

	if imageNameRef.Identifier() == "" && tags.SHAPattern.MatchString(imageName) {
		// image is a digest and was parsed as repository-only reference
		digest = imageName
	} else if imageNameRef.Context().String() != "" {
		digest = imageNameRef.Context().Digest(digest).Name()
	}

	signatureSourceRef := imageNameRef

	var resolvedNameRef name.Reference
	if resolvedName != "" {
		// use resolved name for signature verification -> potentially get signature from internal registry
		resolvedNameRefUsed, err := images.GetImageReference(ctx, c, namespace, resolvedName)
		if err != nil {
			return fmt.Errorf("error parsing image reference %s: %w", resolvedName, err)
		}
		signatureSourceRef = resolvedNameRefUsed

		// for pattern matching we use the reference without any defaults
		resolvedNameRef, err = name.ParseReference(resolvedName, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
		if err != nil {
			return fmt.Errorf("error parsing image reference %s: %w", resolvedName, err)
		}
	}

	// > NamePatterns
	imagenameCovered := nameselector.ImageCovered(imageNameRef, digest, selector.NamePatterns)
	resolvedNameCovered := resolvedNameRef != nil && nameselector.ImageCovered(resolvedNameRef, digest, selector.NamePatterns)
	if !imagenameCovered && !resolvedNameCovered { // could be the same check twice here or the latter could be the resolvedNameRef
		return &ImageSelectorNoMatchError{ImageName: imageName, Field: "namePatterns", Err: fmt.Errorf("Neither image [%s] nor resolved name [%s] match name patterns: %v", imageName, resolvedName, selector.NamePatterns)}
	}

	// > Signatures
	// Any verification error or failed verification issue will error out
	for _, rule := range selector.Signatures {
		if err := signatureselector.VerifySignatureRule(ctx, c, namespace, signatureSourceRef.String(), rule, opts.SignatureOpts, remoteOpts...); err != nil {
			return &ImageSelectorNoMatchError{ImageName: imageName, Field: "signatures", Err: err}
		}
	}
	return nil
}
