package cosign

import (
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func FindSignatureImage(imageRef name.Reference, opts ...remote.Option) (name.Tag, ggcrv1.Image, error) {
	if digest, ok := imageRef.(name.Digest); ok {
		tag, hash, err := FindSignature(digest, opts...)
		if err != nil {
			return name.Tag{}, nil, err
		}
		if hash.Hex == "" {
			return name.Tag{}, nil, nil
		}

		img, err := remote.Image(tag, opts...)

		return tag, img, err
	} else {
		digeststr, err := SimpleDigest(imageRef, opts...)
		if err != nil {
			return name.Tag{}, nil, err
		}
		return FindSignatureImage(imageRef.Context().Digest(digeststr), opts...)
	}
}
