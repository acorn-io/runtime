package apps

import (
	"context"
	"fmt"
	"strings"

	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func (s *Storage) checkRemotePermissions(ctx context.Context, namespace, image string) error {
	keyChain, err := pullsecret.Keychain(ctx, s.client, namespace)
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	_, err = remote.Image(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(keyChain))
	if err != nil {
		return fmt.Errorf("failed to pull %s: %v", image, err)
	}
	return nil
}

func (s *Storage) resolveTag(ctx context.Context, namespace, image string) (string, error) {
	localImage, err := s.images.ImageGet(ctx, image)
	if apierror.IsNotFound(err) {
		if tags.IsLocalReference(image) {
			return "", err
		}
		if err := s.checkRemotePermissions(ctx, namespace, image); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else {
		return strings.TrimPrefix(localImage.Digest, "sha256:"), nil
	}
	return image, nil
}
