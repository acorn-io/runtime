package run

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
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

func validProto(p string) (v1.Protocol, bool) {
	ret := v1.Protocol(p)
	switch ret {
	case v1.ProtocolTCP:
		fallthrough
	case v1.ProtocolUDP:
		fallthrough
	case v1.ProtocolHTTP:
		fallthrough
	case v1.ProtocolAll:
		fallthrough
	case v1.ProtocolNone:
		return ret, true
	}
	return ret, false
}

func ParsePorts(args []string) (result []v1.PortBinding, protos []v1.Protocol, _ error) {
	for _, arg := range args {
		if p, ok := validProto(arg); ok {
			protos = append(protos, p)
			continue
		}

		port, proto, _ := strings.Cut(arg, "/")
		port, targetPort, ok := strings.Cut(port, ":")
		if !ok {
			targetPort = port
		}
		targetPort = strings.TrimSpace(targetPort)
		port = strings.TrimSpace(port)
		if targetPort == "" || port == "" {
			return nil, nil, fmt.Errorf("invalid service binding [%s] must not have zero length value", arg)
		}
		iTargetPort, err := strconv.Atoi(targetPort)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port %s: %w", targetPort, err)
		}
		iPort, err := strconv.Atoi(port)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port %s: %w", port, err)
		}
		result = append(result, v1.PortBinding{
			TargetPort: int32(iTargetPort),
			Port:       int32(iPort),
			Protocol:   v1.Protocol(proto),
		})
	}
	return
}

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
	Name             string
	GenerateName     string
	Namespace        string
	Annotations      map[string]string
	Labels           map[string]string
	Endpoints        []v1.EndpointBinding
	Volumes          []v1.VolumeBinding
	Secrets          []v1.SecretBinding
	Services         []v1.ServiceBinding
	PublishProtocols []v1.Protocol
	Profiles         []string
	Ports            []v1.PortBinding
	DeployArgs       map[string]interface{}
	DevMode          *bool
	Client           client.WithWatch
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
			Ports:            opts.Ports,
			Image:            image,
			Endpoints:        opts.Endpoints,
			Volumes:          opts.Volumes,
			Secrets:          opts.Secrets,
			Services:         opts.Services,
			DeployArgs:       opts.DeployArgs,
			PublishProtocols: opts.PublishProtocols,
			Profiles:         opts.Profiles,
			DevMode:          opts.DevMode,
		},
	}

	if len(app.Spec.PublishProtocols) == 0 && len(app.Spec.Ports) == 0 {
		cfg, err := config.Get(ctx, opts.Client)
		if err != nil {
			return nil, err
		}
		for _, protocol := range cfg.PublishProtocolsByDefault {
			app.Spec.PublishProtocols = append(app.Spec.PublishProtocols, v1.Protocol(protocol))
		}
	}

	return app, opts.Client.Create(ctx, app)
}
