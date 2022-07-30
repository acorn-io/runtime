package run

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/goombaio/namegenerator"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

type Options struct {
	Name         string
	GenerateName string
	Namespace    string
	Annotations  map[string]string
	Labels       map[string]string
	PublishMode  v1.PublishMode
	Volumes      []v1.VolumeBinding
	Secrets      []v1.SecretBinding
	Links        []v1.ServiceBinding
	Profiles     []string
	Ports        []v1.PortBinding
	DeployArgs   map[string]interface{}
	DevMode      *bool
	Client       client.WithWatch
	Permissions  *v1.Permissions
}

func (o *Options) Complete() (*Options, error) {
	var (
		opts Options
		err  error
	)
	if o != nil {
		opts = *o
	}

	if opts.Name == "" && opts.GenerateName == "" {
		opts.Name = NameGenerator.Generate()
	}

	if opts.Namespace == "" {
		opts.Namespace = system.UserNamespace()
	}

	if opts.Client == nil {
		opts.Client, err = hclient.Default()
		if err != nil {
			return nil, err
		}
	}

	return &opts, nil
}

func createNamespace(ctx context.Context, c client.Client, name string) error {
	ns := &corev1.Namespace{}
	err := c.Get(ctx, hclient.ObjectKey{
		Name: name,
	}, ns)
	if apierror.IsNotFound(err) {
		err := c.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
		if err != nil {
			return fmt.Errorf("unable to create namespace %s: %w", name, err)
		}
		return nil
	}
	return err
}

func Run(ctx context.Context, image string, opts *Options) (*v1.AppInstance, error) {
	opts, err := opts.Complete()
	if err != nil {
		return nil, err
	}

	if err := createNamespace(ctx, opts.Client, opts.Namespace); err != nil {
		return nil, err
	}

	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:         opts.Name,
			GenerateName: opts.GenerateName,
			Namespace:    opts.Namespace,
			Labels:       opts.Labels,
			Annotations:  opts.Annotations,
		},
		Spec: v1.AppInstanceSpec{
			Ports:       opts.Ports,
			Image:       image,
			Volumes:     opts.Volumes,
			Secrets:     opts.Secrets,
			Links:       opts.Links,
			DeployArgs:  opts.DeployArgs,
			Profiles:    opts.Profiles,
			PublishMode: opts.PublishMode,
			DevMode:     opts.DevMode,
			Permissions: opts.Permissions,
		},
	}

	if app.Labels == nil {
		app.Labels = map[string]string{}
	}

	app.Labels[labels.AcornRootNamespace] = app.Namespace
	app.Labels[labels.AcornManaged] = "true"
	return app, opts.Client.Create(ctx, app)
}
