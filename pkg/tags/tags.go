package tags

import (
	"context"
	"regexp"
	"strings"

	"github.com/acorn-io/baaah/pkg/uncached"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/google/go-containerregistry/pkg/name"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	SHAPermissivePrefixPattern = regexp.MustCompile(`^[a-f\d]{3,64}$`)
	SHAPattern                 = regexp.MustCompile(`^[a-f\d]{64}$`)
	DigestPattern              = regexp.MustCompile(`^sha256:[a-f\d]{64}$`)
	// Can't use the NoDefaultRegistry const from the images packages without causing a dependency cycle
	noDefaultRegistry = "xxx-no-reg"
)

func IsImageDigest(s string) bool {
	return DigestPattern.MatchString(s)
}

func IsLocalReference(image string) bool {
	if strings.HasPrefix(image, "sha256:") {
		return true
	}
	if SHAPermissivePrefixPattern.MatchString(image) {
		return true
	}
	return false
}

// HasNoSpecifiedRegistry returns true if there is no registry specified in the image name, or if an error occurred
// while trying to parse it into a reference.
func HasNoSpecifiedRegistry(image string) bool {
	ref, err := name.ParseReference(image, name.WithDefaultRegistry(noDefaultRegistry))
	return err != nil || ref.Context().RegistryStr() == noDefaultRegistry
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
func GetTagsMatchingRepository(ctx context.Context, reference name.Reference, c client.Reader, namespace, defaultReg string) ([]string, error) {
	images, err := Get(ctx, c, namespace)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, image := range images.Items {
		if image.Remote {
			continue
		}
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
	// use apiv1.Image here so that get logic does the resolution of names, tags, digests etc
	localImage := &apiv1.Image{}

	err := c.Get(ctx, kclient.ObjectKey{
		Name:      strings.ReplaceAll(image, "/", "+"),
		Namespace: namespace,
	}, uncached.Get(localImage))

	if apierrors.IsNotFound(err) {
		if IsLocalReference(image) {
			return "", false, err
		}
		return image, false, nil
	} else if err != nil {
		return "", false, err
	}
	return strings.TrimPrefix(localImage.Digest, "sha256:"), !localImage.Remote, nil
}
