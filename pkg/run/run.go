package run

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/goombaio/namegenerator"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	nameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

func ParseLinks(args []string) (result []v1.ServiceBinding, _ error) {
	for _, arg := range args {
		existing, secName, ok := strings.Cut(arg, ":")
		if !ok {
			secName = existing
		}
		secName = strings.TrimSpace(secName)
		existing = strings.TrimSpace(existing)
		if secName == "" || existing == "" {
			return nil, fmt.Errorf("invalid service binding [%s] must not have zero length value", arg)
		}
		result = append(result, v1.ServiceBinding{
			Target:  secName,
			Service: existing,
		})
	}
	return
}

func ParseSecrets(args []string) (result []v1.SecretBinding, _ error) {
	for _, arg := range args {
		existing, secName, ok := strings.Cut(arg, ":")
		if !ok {
			secName = existing
		}
		secName = strings.TrimSpace(secName)
		existing = strings.TrimSpace(existing)
		if secName == "" || existing == "" {
			return nil, fmt.Errorf("invalid endpoint binding [%s] must not have zero length value", arg)
		}
		result = append(result, v1.SecretBinding{
			Secret:        existing,
			SecretRequest: secName,
		})
	}
	return
}

func ParseVolumes(args []string) (result []v1.VolumeBinding, _ error) {
	for _, arg := range args {
		existing, volName, ok := strings.Cut(arg, ":")
		if !ok {
			volName = existing
		}
		volName = strings.TrimSpace(volName)
		existing = strings.TrimSpace(existing)
		if volName == "" || existing == "" {
			return nil, fmt.Errorf("invalid endpoint binding [%s] must not have zero length value", arg)
		}
		result = append(result, v1.VolumeBinding{
			Volume:        existing,
			VolumeRequest: volName,
		})
	}
	return
}

func ParseEndpoints(args []string) (result []v1.EndpointBinding, _ error) {
	for _, arg := range args {
		public, private, ok := strings.Cut(arg, ":")
		if !ok {
			private = public
		}
		private = strings.TrimSpace(private)
		public = strings.TrimSpace(public)
		if private == "" || public == "" {
			return nil, fmt.Errorf("invalid endpoint binding [%s] must not have zero length value", arg)
		}
		result = append(result, v1.EndpointBinding{
			Target:   private,
			Hostname: public,
		})
	}
	return
}

type Options struct {
	Name         string
	GenerateName string
	Namespace    string
	Annotations  map[string]string
	Labels       map[string]string
	Endpoints    []v1.EndpointBinding
	Volumes      []v1.VolumeBinding
	Secrets      []v1.SecretBinding
	Services     []v1.ServiceBinding
	DeployParams map[string]interface{}
	Client       client.WithWatch
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
			Name:         opts.Name,
			GenerateName: opts.GenerateName,
			Namespace:    opts.Namespace,
			Labels:       opts.Labels,
			Annotations:  opts.Annotations,
		},
		Spec: v1.AppInstanceSpec{
			PublishAllPorts: true,
			Image:           image,
			Endpoints:       opts.Endpoints,
			Volumes:         opts.Volumes,
			Secrets:         opts.Secrets,
			Services:        opts.Services,
			DeployParams:    opts.DeployParams,
		},
	}

	return app, opts.Client.Create(ctx, app)
}
