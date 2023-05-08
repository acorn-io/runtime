package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *DefaultClient) ImageTag(ctx context.Context, imageName, tag string) error {
	image, err := c.ImageGet(ctx, imageName)
	if apierrors.IsNotFound(err) {
		return err
	}
	tagResult := &apiv1.ImageTag{}
	err = c.RESTClient.Post().
		Namespace(image.Namespace).
		Resource("images").
		Name(image.Name).
		SubResource("tag").
		Body(&apiv1.ImageTag{
			Tag: tag,
		}).Do(ctx).Into(tagResult)
	return err
}

func (c *DefaultClient) ImageDetails(ctx context.Context, imageName string, opts *ImageDetailsOptions) (*ImageDetails, error) {
	imageName = strings.ReplaceAll(imageName, "/", "+")

	detailsResult := &apiv1.ImageDetails{}

	if opts != nil {
		detailsResult.DeployArgs = opts.DeployArgs
		detailsResult.Profiles = opts.Profiles
		detailsResult.NestedDigest = opts.NestedDigest
		detailsResult.Auth = opts.Auth
	}

	err := c.RESTClient.Post().
		Namespace(c.Namespace).
		Resource("images").
		Name(imageName).
		SubResource("details").
		Body(detailsResult).
		Do(ctx).Into(detailsResult)
	if err != nil {
		return nil, err
	}

	return &ImageDetails{
		AppImage:   detailsResult.AppImage,
		AppSpec:    detailsResult.AppSpec,
		Params:     detailsResult.Params,
		ParseError: detailsResult.ParseError,
	}, nil
}

func (c *DefaultClient) ImagePull(ctx context.Context, imageName string, opts *ImagePullOptions) (<-chan ImageProgress, error) {
	body := &apiv1.ImagePull{}
	if opts != nil {
		body.Auth = opts.Auth
	}

	url := c.RESTClient.Get().
		Namespace(c.Namespace).
		Resource("images").
		Name(strings.ReplaceAll(imageName, "/", "+")).
		SubResource("pull").
		URL()

	conn, _, err := c.Dialer.DialWebsocket(ctx, url.String(), nil)
	if err != nil {
		return nil, err
	}

	if err := conn.WriteJSON(body); err != nil {
		return nil, err
	}

	result := make(chan ImageProgress, 1000)
	go func() {
		defer close(result)
		defer conn.Close()
		for {
			_, data, err := conn.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			} else if err != nil {
				result <- ImageProgress{
					Error: err.Error(),
				}
				break
			}

			progress := ImageProgress{}
			if err := json.Unmarshal(data, &progress); err == nil {
				result <- progress
			} else {
				result <- ImageProgress{
					Error: err.Error(),
				}
			}
		}
	}()

	return result, nil
}

func (c *DefaultClient) ImagePush(ctx context.Context, imageName string, opts *ImagePushOptions) (<-chan ImageProgress, error) {
	body := &apiv1.ImagePush{}
	if opts != nil {
		body.Auth = opts.Auth
	}

	image, err := c.ImageGet(ctx, imageName)
	if err != nil {
		return nil, err
	}

	url := c.RESTClient.Get().
		Namespace(image.Namespace).
		Resource("images").
		Name(strings.ReplaceAll(imageName, "/", "+")).
		SubResource("push").
		URL()

	conn, _, err := c.Dialer.DialWebsocket(ctx, url.String(), nil)
	if err != nil {
		return nil, err
	}

	if err := conn.WriteJSON(body); err != nil {
		return nil, err
	}

	result := make(chan ImageProgress)
	go func() {
		defer close(result)
		defer conn.Close()
		for {
			_, data, err := conn.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			} else if err != nil {
				result <- ImageProgress{
					Error: err.Error(),
				}
				break
			}

			progress := ImageProgress{}
			if err := json.Unmarshal(data, &progress); err == nil {
				result <- progress
			} else {
				result <- ImageProgress{
					Error: err.Error(),
				}
			}
		}
	}()

	return result, nil
}

func (c *DefaultClient) ImageDelete(ctx context.Context, imageName string, opts *ImageDeleteOptions) (*apiv1.Image, error) {
	image, err := c.ImageGet(ctx, imageName)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var remainingTags []string

	imageParsedRef, err := name.ParseReference(imageName, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
	if err != nil {
		return nil, err
	}
	tagToDelete := ""
	repoToDelete := ""

	if _, ok := imageParsedRef.(name.Digest); ok {
		// if the image is referenced by digest, we need to delete the tag with only registry/repository
		repoToDelete = imageParsedRef.Context().Name()
	} else if tag, err := name.NewTag(imageName, name.StrictValidation); err == nil {
		tagToDelete = tag.Name()
	}

	// Getting an image, auto-suffixed with :latest also returns images that don't have that tag (explicit :latest) at all, potentially yielding errors down the line
	if tagToDelete != "" && !slices.Contains(image.Tags, tagToDelete) {
		return image, fmt.Errorf("image tag %s not found", imageName)
	}

	if len(image.Tags) == 1 {
		return image, c.Client.Delete(ctx, image)
	}

	for _, tag := range image.Tags {
		if tag == tagToDelete {
			logrus.Debugf("Delete: %s", tag)
			continue
		}

		ref, err := name.ParseReference(tag, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
		if err != nil {
			return nil, err
		}
		if ref.Context().Name() == repoToDelete {
			logrus.Infof("Delete: %s", tag)
			continue
		}

		// not filtered out, keep it safe
		remainingTags = append(remainingTags, tag)
	}
	if len(remainingTags) != len(image.Tags) {
		image.Tags = remainingTags
		err = c.RESTClient.Put().
			Namespace(image.Namespace).
			Resource("images").
			Name(image.Name).
			Body(image).
			Do(ctx).Into(image)
		return image, err
	}
	if !opts.Force && len(image.Tags) > 1 {
		return nil, fmt.Errorf("unable to delete %s (must be forced) - image is referenced in multiple repositories", imageName)
	}
	return image, c.Client.Delete(ctx, image)
}

func (c *DefaultClient) ImageGet(ctx context.Context, imageName string) (*apiv1.Image, error) {
	result := &apiv1.Image{}
	return result, c.Client.Get(ctx, kclient.ObjectKey{
		Name:      strings.ReplaceAll(imageName, "/", "+"),
		Namespace: c.Namespace,
	}, result)
}

func (c *DefaultClient) ImageList(ctx context.Context) ([]apiv1.Image, error) {
	result := &apiv1.ImageList{}
	err := c.Client.List(ctx, result, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}
