package tags

import (
	"context"
	"regexp"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/uncached"
	"github.com/google/go-containerregistry/pkg/name"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	SHAPermissivePrefixPattern = regexp.MustCompile(`^[a-f\d]{3,64}$`)
	SHAPattern                 = regexp.MustCompile(`^[a-f\d]{64}$`)
)

func IsLocalReference(image string) bool {
	if strings.HasPrefix(image, "sha256:") {
		return true
	}
	if SHAPermissivePrefixPattern.MatchString(image) {
		return true
	}
	return false
}

func Get(ctx context.Context, c client.Reader, namespace string) (apiv1.ImageList, error) {
	var imageList apiv1.ImageList
	if namespace == "" {
		return imageList, nil
	}

	return imageList, c.List(ctx, &imageList, kclient.InNamespace(namespace))
}

// GetTagsMatchingRepository returns the tag portion of local images that match the repository in the supplied reference
// Note that the other functions in this package generally operate on the entire reference of an image as one opaque "tag"
// but this function is only returning the portion that follows the final semicolon. For example, the "tags" returned by the
// Get function are like "foo/bar:v1", but this function would just return "v1" for that image.
func GetTagsMatchingRepository(reference name.Reference, ctx context.Context, c client.Reader, namespace, defaultReg string) ([]string, error) {
	images, err := Get(ctx, c, namespace)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, image := range images.Items {
		for _, tag := range image.Tags {
			r, err := name.ParseReference(tag, name.WithDefaultRegistry(defaultReg))
			if err != nil {
				continue
			}
			if r.Context() == reference.Context() {
				result = append(result, r.Identifier())
			}
		}
	}
	return result, nil
}

// ResolveLocal determines if the image is local and if it is, resolves it to an image ID that can be pulled from the
// local registry
func ResolveLocal(ctx context.Context, c kclient.Client, namespace, image string) (string, bool, error) {
	localImage := &apiv1.Image{}

	err := c.Get(ctx, kclient.ObjectKey{
		Name:      strings.ReplaceAll(image, "/", "+"),
		Namespace: namespace,
	}, uncached.Get(localImage))

	if apierrors.IsNotFound(err) {
		if IsLocalReference(image) {
			return "", false, err
		}
	} else if err != nil {
		return "", false, err
	} else {
		return strings.TrimPrefix(localImage.Digest, "sha256:"), true, nil
	}
	return image, false, nil
}
