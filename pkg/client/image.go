package client

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *client) ImageTag(ctx context.Context, imageName, tag string) error {
	image, err := c.ImageGet(ctx, imageName)
	if err != nil {
		return err
	}

	tagResult := &apiv1.ImageTag{}
	err = c.RESTClient.Post().
		Namespace(image.Namespace).
		Resource("images").
		Name(image.Name).
		SubResource("tag").
		Body(&apiv1.ImageTag{
			TagName: tag,
		}).Do(ctx).Into(tagResult)
	return err
}

func (c *client) ImageDetails(ctx context.Context, imageName string, opts *ImageDetailsOptions) (*ImageDetails, error) {
	imageName = strings.ReplaceAll(imageName, "/", "+")

	if opts == nil {
		opts = &ImageDetailsOptions{}
	}

	detailsResult := &apiv1.ImageDetails{}
	err := c.RESTClient.Post().
		Namespace(c.Namespace).
		Resource("images").
		Name(imageName).
		SubResource("details").
		Body(&apiv1.ImageDetails{
			PullSecrets: opts.PullSecrets,
		}).Do(ctx).Into(detailsResult)
	if err != nil {
		return nil, err
	}

	return &ImageDetails{
		AppImage: detailsResult.AppImage,
	}, nil
}

func (c *client) ImagePull(ctx context.Context, imageName string, opts *ImagePullOptions) (<-chan ImageProgress, error) {
	if opts == nil {
		opts = &ImagePullOptions{}
	}

	resp, err := c.RESTClient.Post().
		Namespace(c.Namespace).
		Resource("images").
		Name(strings.ReplaceAll(imageName, "/", "+")).
		SubResource("pull").
		Body(&apiv1.ImagePull{
			PullSecrets: opts.PullSecrets,
		}).
		Stream(ctx)
	if err != nil {
		return nil, err
	}

	result := make(chan ImageProgress, 1000)
	go func() {
		defer close(result)
		lines := bufio.NewScanner(resp)
		for lines.Scan() {
			line := lines.Text()
			progress := ImageProgress{}
			if err := json.Unmarshal([]byte(line), &progress); err == nil {
				result <- progress
			} else {
				result <- ImageProgress{
					Error: err.Error(),
				}
			}
		}

		err := lines.Err()
		if err != nil {
			result <- ImageProgress{
				Error: err.Error(),
			}
		}
	}()

	return result, nil
}

func (c *client) ImagePush(ctx context.Context, imageName string, opts *ImagePushOptions) (<-chan ImageProgress, error) {
	image, err := c.ImageGet(ctx, imageName)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ImagePushOptions{}
	}

	resp, err := c.RESTClient.Post().
		Namespace(image.Namespace).
		Resource("images").
		Name(strings.ReplaceAll(imageName, "/", "+")).
		SubResource("push").
		Body(&apiv1.ImagePush{
			PullSecrets: opts.PullSecrets,
		}).
		Stream(ctx)
	if err != nil {
		return nil, err
	}

	result := make(chan ImageProgress)
	go func() {
		defer close(result)
		lines := bufio.NewScanner(resp)
		for lines.Scan() {
			line := lines.Text()
			progress := ImageProgress{}
			if err := json.Unmarshal([]byte(line), &progress); err == nil {
				result <- progress
			} else {
				result <- ImageProgress{
					Error: err.Error(),
				}
			}
		}

		err := lines.Err()
		if err != nil {
			result <- ImageProgress{
				Error: err.Error(),
			}
		}
	}()

	return result, nil
}

func (c *client) ImageDelete(ctx context.Context, imageName string) (*apiv1.Image, error) {
	image, err := c.ImageGet(ctx, imageName)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return image, c.Client.Delete(ctx, image)
}

func (c *client) ImageGet(ctx context.Context, imageName string) (*apiv1.Image, error) {
	result := &apiv1.Image{}
	return result, c.Client.Get(ctx, kclient.ObjectKey{
		Name:      strings.ReplaceAll(imageName, "/", "+"),
		Namespace: c.Namespace,
	}, result)
}

func (c *client) ImageList(ctx context.Context) ([]apiv1.Image, error) {
	result := &apiv1.ImageList{}

	err := c.Client.List(ctx, result, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}
