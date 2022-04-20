package client

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/build/buildkit"
	"github.com/ibuildthecloud/herd/pkg/pull"
	"github.com/ibuildthecloud/herd/pkg/remoteopts"
	tags2 "github.com/ibuildthecloud/herd/pkg/tags"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (c *client) Tag(ctx context.Context, imageName, tag string) (*Image, error) {
	fullTag, err := name.NewTag(tag)
	if err != nil {
		return nil, err
	}

	image, err := c.ImageGet(ctx, imageName)
	if err != nil {
		return nil, err
	}

	return image, tags2.Write(ctx, c.Client, c.Namespace, image.Digest, fullTag.Name())
}

func (c *client) GetAppImage(ctx context.Context, imageName string, pullSecrets []string) (*v1.AppImage, error) {
	imageName, err := c.resolveTag(ctx, imageName, pullSecrets)
	if err != nil {
		return nil, err
	}
	return pull.AppImage(ctx, c.Client, c.Namespace, imageName, pullSecrets)
}

func (c *client) ImagePull(ctx context.Context, imageName string) (*Image, error) {
	writeOpts, err := remoteopts.GetRemoteWriteOptions(ctx, c.Client)
	if err != nil {
		return nil, err
	}

	opts, err := remoteopts.GetRemoteOptions(ctx, c.Client)
	if err != nil {
		return nil, err
	}

	pullTag, err := name.ParseReference(imageName)
	if err != nil {
		return nil, err
	}

	index, err := remote.Index(pullTag, writeOpts...)
	if err != nil {
		return nil, err
	}

	hash, err := index.Digest()
	if err != nil {
		return nil, err
	}

	repo, err := c.getRepo()
	if err != nil {
		return nil, err
	}

	if err := remote.WriteIndex(repo.Digest(hash.Hex), index, opts...); err != nil {
		return nil, err
	}

	return c.Tag(ctx, hash.Hex, imageName)
}

func (c *client) ImagePush(ctx context.Context, imageName string) (*Image, error) {
	pushTag, err := name.NewTag(imageName)
	if err != nil {
		return nil, err
	}

	image, err := c.ImageGet(ctx, imageName)
	if err != nil {
		return nil, err
	}

	opts, err := remoteopts.GetRemoteOptions(ctx, c.Client)
	if err != nil {
		return nil, err
	}

	repo, err := c.getRepo()
	if err != nil {
		return nil, err
	}

	remoteImage, err := remote.Index(repo.Digest(image.Digest), opts...)
	if err != nil {
		return nil, err
	}

	writeOpts, err := remoteopts.GetRemoteWriteOptions(ctx, c.Client)
	if err != nil {
		return nil, err
	}

	return image, remote.WriteIndex(pushTag, remoteImage, writeOpts...)
}

func (c *client) ImageDelete(ctx context.Context, imageName string) (*Image, error) {
	image, tagName, err := c.imageGet(ctx, imageName)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	if tagName != "" && len(image.Tags) > 1 {
		return image, tags2.Remove(ctx, c.Client, c.Namespace, image.Digest, tagName)
	}

	repo, err := c.getRepo()
	if err != nil {
		return nil, err
	}

	opts, err := remoteopts.GetRemoteOptions(ctx, c.Client)
	if err != nil {
		return nil, err
	}

	return image, remote.Delete(repo.Digest(image.Digest), opts...)
}

func (c *client) ImageGet(ctx context.Context, imageName string) (*Image, error) {
	if ok, err := buildkit.Exists(ctx, c.Client); err != nil {
		return nil, err
	} else if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    "herd.project.io",
			Resource: "images",
		}, imageName)
	}

	image, _, err := c.imageGet(ctx, imageName)
	return image, err
}

func (c *client) imageGet(ctx context.Context, image string) (*Image, string, error) {
	images, err := c.ImageList(ctx)
	if err != nil {
		return nil, "", err
	}

	var (
		digest       string
		digestPrefix string
		tagName      string
	)

	if strings.HasPrefix(image, "sha256:") {
		digest = image
	} else if tags2.SHAPattern.MatchString(image) {
		digest = "sha256:" + image
	} else if tags2.SHAShortPattern.MatchString(image) {
		digestPrefix = "sha256:" + image
	} else {
		tag, err := name.ParseReference(image)
		if err != nil {
			return nil, "", err
		}
		tagName = tag.Name()
	}

	for _, image := range images {
		if image.Digest == digest {
			return &image, "", nil
		} else if digestPrefix != "" && strings.HasPrefix(image.Digest, digestPrefix) {
			return &image, "", nil
		}
		for _, imageTag := range image.Tags {
			imageParsedTag, err := name.NewTag(imageTag)
			if err != nil {
				continue
			}
			if imageParsedTag.Name() == tagName {
				return &image, imageTag, nil
			}
		}
	}

	return nil, "", apierrors.NewNotFound(schema.GroupResource{
		Group:    "herd-project.io",
		Resource: "images",
	}, image)
}

func (c *client) getRepo() (name.Repository, error) {
	return name.NewRepository("127.0.0.1:5000/herd/" + c.Namespace)
}

func (c *client) ImageList(ctx context.Context) (result []Image, _ error) {
	if ok, err := buildkit.Exists(ctx, c.Client); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	opts, err := remoteopts.GetRemoteOptions(ctx, c.Client)
	if err != nil {
		return nil, err
	}

	repo, err := c.getRepo()
	if err != nil {
		return nil, err
	}

	names, err := remote.List(repo, opts...)
	if tErr, ok := err.(*transport.Error); ok && tErr.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	tags, err := tags2.Get(ctx, c.Client, c.Namespace)
	if err != nil {
		return nil, err
	}

	for _, name := range names {
		if !tags2.SHAPattern.MatchString(name) {
			continue
		}
		result = append(result, Image{
			Digest: "sha256:" + name,
			Tags:   tags[name],
		})
	}

	return result, nil
}
