package images

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"

	tags2 "github.com/acorn-io/runtime/pkg/tags"
)

type ErrImageNotFound struct {
	ImageSearch string
}

func (e ErrImageNotFound) Error() string {
	return fmt.Sprintf("image not found: %s", e.ImageSearch)
}

type ErrImageIdentifierNotUnique struct {
	ImageSearch string
}

func (e ErrImageIdentifierNotUnique) Error() string {
	return fmt.Sprintf("image identifier not unique: %s", e.ImageSearch)
}

// findImageMatch matches images by digest, digest prefix, or tag name:
//
// - digest (raw): sha256:<digest> or <digest> (exactly 64 chars)
// - digest (image): <registry>/<repo>@sha256:<digest> or <repo>@sha256:<digest>
// - digest prefix: sha256:<digest prefix> (min. 3 chars)
// - tag name: <registry>/<repo>:<tag> or <repo>:<tag>
// - tag name (with default): <registry>/<repo> or <repo> -> Will be matched against the default tag (:latest)
//   - Note: if we get some string here, that matches the SHAPermissivePrefixPattern, it could be both a digest or a name without a tag
//     so we will try to match it against the default tag (:latest) first and if that fails, we treat it as a digest(-prefix)
func FindImageMatch(images apiv1.ImageList, search string) (*apiv1.Image, string, error) {
	var (
		repoDigest     name.Digest
		digest         string
		digestPrefix   string
		tagName        string
		tagNameDefault string
		canBeMultiple  bool // if true, we will not return on first match
	)
	if strings.HasPrefix(search, "sha256:") {
		digest = search
	} else if tags2.SHAPattern.MatchString(search) {
		digest = "sha256:" + search
		tagNameDefault = search // this could as well be some name without registry/repo path and tag
	} else if tags2.SHAPermissivePrefixPattern.MatchString(search) {
		digestPrefix = "sha256:" + search
		tagNameDefault = search // this could as well be some name without registry/repo path and tag
	} else {
		ref, err := name.ParseReference(search, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
		if err != nil {
			return nil, "", err
		}
		if ref.Identifier() == "" {
			tagNameDefault = ref.Name() // some name without a tag, so we will try to match it against the default tag (:latest)
			canBeMultiple = true
		} else if dig, ok := ref.(name.Digest); ok {
			repoDigest = dig
		} else {
			tagName = ref.Name()
		}
	}

	if tagNameDefault != "" {
		// add default tag (:latest)
		t, err := name.ParseReference(tagNameDefault, name.WithDefaultRegistry(""))
		if err != nil {
			return nil, "", err
		}
		tagNameDefault = t.Name()
	}

	var matchedImage apiv1.Image
	var matchedTag string
	for _, image := range images.Items {
		// >>> match by tag name with default tag (:latest)
		if tagNameDefault != "" {
			for _, tag := range image.Tags {
				if tag == tagNameDefault {
					return &image, tag, nil
				}
			}
		}

		// >>> match by digest or digest prefix
		if image.Digest == digest {
			return &image, "", nil
		} else if digestPrefix != "" && strings.HasPrefix(image.Digest, digestPrefix) {
			if matchedImage.Digest != "" && matchedImage.Digest != image.Digest {
				return nil, "", ErrImageIdentifierNotUnique{ImageSearch: search}
			}
			matchedImage = image
		}

		// >>> match by repo digest
		// this returns an image which matches the digest and has at least one tag
		// which matches the repo part of the repo digest.
		if repoDigest.Name() != "" && image.Digest == repoDigest.DigestStr() {
			for _, tag := range image.Tags {
				imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
				if err != nil {
					continue
				}
				if imageParsedTag.Context().Name() == repoDigest.Context().Name() {
					return &image, tag, nil
				}
			}
		}

		// >>> match by tag name
		for _, tag := range image.Tags {
			if tag == search {
				if !canBeMultiple {
					return &image, tag, nil
				}
				if matchedImage.Digest != "" && matchedImage.Digest != image.Digest {
					return nil, "", ErrImageIdentifierNotUnique{ImageSearch: search}
				}
				matchedImage = image
				matchedTag = tag
			} else if tag != "" {
				imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""), name.WithDefaultTag("")) // no default here, as we also have repo-only tag items
				if err != nil {
					continue
				}
				if imageParsedTag.Name() == tagName {
					if !canBeMultiple {
						return &image, tag, nil
					}
					if matchedImage.Digest != "" && matchedImage.Digest != image.Digest {
						return nil, "", ErrImageIdentifierNotUnique{ImageSearch: search}
					}
					matchedImage = image
					matchedTag = tag
				}
			}
		}
	}

	if matchedImage.Digest != "" {
		return &matchedImage, matchedTag, nil
	}

	return nil, "", ErrImageNotFound{ImageSearch: search}
}
