package run

import (
	"context"
	"fmt"
	"time"

	"github.com/goombaio/namegenerator"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	hclient "github.com/ibuildthecloud/herd/pkg/client"
	"github.com/ibuildthecloud/herd/pkg/system"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	nameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

type Options struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
	Client      client.Client
}

func (o *Options) Complete() (*Options, error) {
	var (
		opts Options
		err  error
	)
	if o != nil {
		opts = *o
	}

	if opts.Name == "" {
		opts.Name = nameGenerator.Generate()
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
			Name:        opts.Name,
			Namespace:   opts.Namespace,
			Labels:      opts.Labels,
			Annotations: opts.Annotations,
		},
		Spec: v1.AppInstanceSpec{
			Image: image,
		},
		Status: v1.AppInstanceStatus{},
	}

	return app, opts.Client.Create(ctx, app)
}
