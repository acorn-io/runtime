package autoupgrade

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/imageallowrules"
	"github.com/acorn-io/acorn/pkg/images"
	tags2 "github.com/acorn-io/acorn/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type daemonClient interface {
	getConfig(context.Context) (*apiv1.Config, error)
	listAppInstances(context.Context) ([]v1.AppInstance, error)
	updateAppStatus(context.Context, *v1.AppInstance) error
	listTags(context.Context, string, string, ...remote.Option) ([]string, error)
	getTagsMatchingRepo(context.Context, name.Reference, string, string) ([]string, error)
	imageDigest(context.Context, string, string, ...remote.Option) (string, error)
	resolveLocalTag(context.Context, string, string) (string, bool, error)
	checkImageAllowed(context.Context, string, string) error
}

type client struct {
	client kclient.Client
}

func (c *client) getConfig(ctx context.Context) (*apiv1.Config, error) {
	return config.Get(ctx, c.client)
}

func (c *client) listAppInstances(ctx context.Context) ([]v1.AppInstance, error) {
	var appInstanceList v1.AppInstanceList
	return appInstanceList.Items, c.client.List(ctx, &appInstanceList)
}

func (c *client) updateAppStatus(ctx context.Context, app *v1.AppInstance) error {
	return c.client.Status().Update(ctx, app)
}

func (c *client) listTags(ctx context.Context, namespace, name string, opts ...remote.Option) ([]string, error) {
	_, tags, pullErr := images.ListTags(ctx, c.client, namespace, name, opts...)
	return tags, pullErr
}

func (c *client) getTagsMatchingRepo(ctx context.Context, current name.Reference, namespace, defaultReg string) ([]string, error) {
	return tags2.GetTagsMatchingRepository(ctx, current, c.client, namespace, defaultReg)
}

func (c *client) imageDigest(ctx context.Context, namespace, name string, opts ...remote.Option) (string, error) {
	return images.ImageDigest(ctx, c.client, namespace, name, opts...)
}

func (c *client) resolveLocalTag(ctx context.Context, namespace, name string) (string, bool, error) {
	return tags2.ResolveLocal(ctx, c.client, namespace, name)
}

func (c *client) checkImageAllowed(ctx context.Context, namespace, name string) error {
	return imageallowrules.CheckImageAllowed(ctx, c.client, namespace, name)
}
