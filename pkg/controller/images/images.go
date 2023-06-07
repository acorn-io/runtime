package images

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateImages(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)
	if app.Status.AppImage.ID == "" || app.Status.AppImage.Digest == "" || app.Status.ObservedImageDigest == app.Status.AppImage.Digest {
		// If the image hasn't changed, then don't worry about creating it. It would have been created the first time.
		return nil
	}

	self, err := createImageForSelf(req, app)
	if err != nil {
		return nil
	}

	return createNestedReferences(req, app, self)
}

func createNestedReferences(req router.Request, app *v1.AppInstance, self *v1.ImageInstance) error {
	for _, imageData := range typed.Concat(app.Status.AppImage.ImageData.Acorns, app.Status.AppImage.ImageData.Images) {
		if !tags.IsLocalReference(imageData.Image) {
			continue
		}
		imageName := strings.TrimPrefix(imageData.Image, "sha256:")
		nestedImage := &v1.ImageInstance{}
		err := req.Get(nestedImage, app.Namespace, imageName)
		if err == nil {
			continue
		}

		if apierror.IsNotFound(err) {
			err = req.Client.Create(req.Ctx, &v1.ImageInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imageName,
					Namespace: app.Namespace,
				},
				Remote: self.Remote,
				Repo:   self.Repo,
				Digest: "sha256:" + imageName,
			})
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func createImageForSelf(req router.Request, app *v1.AppInstance) (*v1.ImageInstance, error) {
	var (
		digest    = app.Status.AppImage.Digest
		digestHex = strings.TrimPrefix(digest, "sha256:")
		imageName = app.Status.AppImage.ID
		// update == false means create
		update = true
		image  = &v1.ImageInstance{}
	)

	if tags.IsLocalReference(imageName) {
		return image, req.Get(image, app.Namespace, imageName)
	}

	ref, err := name.ParseReference(imageName, name.WithDefaultTag(""))
	if err != nil {
		return nil, err
	}

	hasTag := true
	// if reference is a digest just store the repo
	if d, ok := ref.(name.Digest); ok {
		imageName = d.Context().String()
		hasTag = false
	} else if t, ok := ref.(name.Tag); ok {
		hasTag = t.TagStr() != ""
	}

	err = req.Get(image, app.Namespace, digestHex)
	if err != nil && !apierror.IsNotFound(err) {
		return nil, err
	} else if apierror.IsNotFound(err) {
		update = false
	}

	if update {
		for _, tag := range image.Tags {
			if tag == imageName {
				// We found the existing image and the tag is there, no need to do anything
				return image, nil
			} else if !hasTag && strings.HasPrefix(tag, imageName+":") {
				// We found a close enough match
				return image, nil
			}
		}
	}

	// remove tag from existing images
	if err := removeTag(req, app, imageName); err != nil {
		return nil, err
	}

	if update {
		tags := []string{imageName}
		for _, tag := range image.Tags {
			if tag == ref.Context().String() {
				// remove the "repo only" tag
				continue
			}
			tags = append(tags, tag)
		}
		image.Tags = tags
		return image, req.Client.Update(req.Ctx, image)
	}

	image = &v1.ImageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      digestHex,
			Namespace: app.Namespace,
		},
		Remote: true,
		Repo:   ref.Context().String(),
		Digest: digest,
		Tags: []string{
			imageName,
		},
	}

	return image, req.Client.Create(req.Ctx, image)
}

func removeTag(req router.Request, app *v1.AppInstance, tag string) error {
	images := &v1.ImageInstanceList{}
	err := req.List(images, &kclient.ListOptions{
		Namespace: app.Namespace,
	})
	if err != nil {
		return err
	}

	for _, image := range images.Items {
		if len(image.Tags) == 0 {
			continue
		}

		newTags := make([]string, 0, len(image.Tags))
		for _, existingTag := range image.Tags {
			if existingTag == tag {
				continue
			}
			newTags = append(newTags, existingTag)
		}

		if len(newTags) != len(image.Tags) {
			image.Tags = newTags
			if err := req.Client.Update(req.Ctx, &image); err != nil {
				return err
			}
		}
	}

	return nil
}
