package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/runtime/pkg/images"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/gorilla/websocket"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/strings/slices"
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
		detailsResult.NoDefaultRegistry = opts.NoDefaultRegistry
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

// ImageDelete handles two use cases: remove a tag from an image or delete an image entirely. Both may go hand in hand when deleting the last remaining tag.
func (c *DefaultClient) ImageDelete(ctx context.Context, imageName string, opts *ImageDeleteOptions) (*apiv1.Image, []string, error) {
	image, err := c.ImageGet(ctx, imageName)
	if err != nil {
		return nil, nil, err
	}

	image, tagToDelete, err := images.FindImageMatch(apiv1.ImageList{Items: []apiv1.Image{*image}}, imageName)
	if err != nil {
		return nil, nil, err
	}

	// Shortcut if there is only one tag
	if len(image.Tags) == 1 {
		return image, image.Tags, c.Client.Delete(ctx, image)
	}

	// If a tag was specified, delete only that tag
	if tagToDelete != "" {
		remainingTags := slices.Filter(nil, image.Tags, func(tag string) bool { return tag != tagToDelete })

		if len(remainingTags) != len(image.Tags) {
			image.Tags = remainingTags
			err = c.RESTClient.Put().
				Namespace(image.Namespace).
				Resource("images").
				Name(image.Name).
				Body(image).
				Do(ctx).Into(image)
			return nil, []string{tagToDelete}, err
		}
	}

	// We only delete an image with >1 tags if the force flag is set
	if !opts.Force && len(image.Tags) > 1 {
		return nil, nil, fmt.Errorf("unable to delete %s (must be forced) - image is referenced in multiple repositories", imageName)
	}

	return image, image.Tags, c.Client.Delete(ctx, image)
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

// FindImage finds an image if exists and returns whether it was found by tag
func FindImage(ctx context.Context, c Client, name string) (*apiv1.Image, string, error) {
	// Early filter autoupgrade patterns, as they will fail lookup since the reference cannot be parsed
	if _, ok := autoupgrade.AutoUpgradePattern(name); ok {
		return nil, "", images.ErrImageNotFound{ImageSearch: name}
	}

	il, err := c.ImageList(ctx)
	if err != nil {
		return nil, "", err
	}

	return images.FindImageMatch(apiv1.ImageList{Items: il}, name)
}
