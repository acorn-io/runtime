package run

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	nameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
	nameRegexp    = regexp.MustCompile("^[a-z][-a-z0-9]+$")
)

func validProto(p string) (v1.Protocol, bool) {
	ret := v1.Protocol(p)
	switch ret {
	case v1.ProtocolTCP:
		fallthrough
	case v1.ProtocolUDP:
		fallthrough
	case v1.ProtocolHTTP:
		return ret, true
	case "":
		return ret, true
	}
	return ret, false
}

func parseNum(str string) (int32, bool, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		if !nameRegexp.MatchString(str) {
			return 0, false, fmt.Errorf("string [%s] does not match %s", str, nameRegexp)
		}
		return 0, false, nil
	}
	return int32(i), true, nil
}

func parseSingle(str string) (int32, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("invalid syntax [%s]: %w", str, err)
	}
	return int32(i), nil
}

func parseQuad(expose bool, left, leftMiddle, rightMiddle, right string) (v1.PortBinding, error) {
	if !expose {
		return v1.PortBinding{}, fmt.Errorf("invalid [%s:%s:%s:%s]: (service:port:service:port) syntax"+
			" is only valid for expose", left, leftMiddle, rightMiddle, right)
	}
	_, leftIsNum, err := parseNum(left)
	if err != nil {
		return v1.PortBinding{}, err
	}
	leftMiddleNum, leftMiddleIsNum, err := parseNum(leftMiddle)
	if err != nil {
		return v1.PortBinding{}, err
	}
	_, rightMiddleIsNum, err := parseNum(rightMiddle)
	if err != nil {
		return v1.PortBinding{}, err
	}
	rightNum, rightIsNum, err := parseNum(right)
	if err != nil {
		return v1.PortBinding{}, err
	}

	if !leftIsNum && leftMiddleIsNum && !rightMiddleIsNum && rightIsNum {
		return v1.PortBinding{
			ServiceName:       left,
			Port:              leftMiddleNum,
			TargetPort:        rightNum,
			TargetServiceName: rightMiddle,
		}, nil
	}
	return v1.PortBinding{}, fmt.Errorf("invalid [%s:%s:%s:%s]: must be (service:port:service:port)",
		left, leftMiddle, rightMiddle, right)
}

func parseTriplet(left, middle, right string) (v1.PortBinding, error) {
	leftNum, leftIsNum, err := parseNum(left)
	if err != nil {
		return v1.PortBinding{}, err
	}
	_, middleIsNum, err := parseNum(middle)
	if err != nil {
		return v1.PortBinding{}, err
	}
	rightNum, rightIsNum, err := parseNum(right)
	if err != nil {
		return v1.PortBinding{}, err
	}

	if leftIsNum && !middleIsNum && rightIsNum {
		// 81:service:80
		return v1.PortBinding{
			Port:              leftNum,
			TargetPort:        rightNum,
			TargetServiceName: middle,
		}, nil
	} else if !leftIsNum && !middleIsNum && rightIsNum {
		// example.com:service:80
		return v1.PortBinding{
			ServiceName:       left,
			TargetPort:        rightNum,
			TargetServiceName: middle,
		}, nil
	}
	return v1.PortBinding{}, fmt.Errorf("invalid binding [%s:%s:%s] must be service:port:targetPort or domain:service:targetPort", left, middle, right)
}

func parseTuple(left, right string) (v1.PortBinding, error) {
	leftNum, leftIsNum, err := parseNum(left)
	if err != nil {
		return v1.PortBinding{}, err
	}
	rightNum, rightIsNum, err := parseNum(right)
	if err != nil {
		return v1.PortBinding{}, err
	}

	if leftIsNum && rightIsNum {
		// 81:80 format
		return v1.PortBinding{
			Port:       leftNum,
			TargetPort: rightNum,
		}, nil
	} else if !leftIsNum && rightIsNum {
		// service:80 format
		return v1.PortBinding{
			TargetPort:        rightNum,
			TargetServiceName: left,
		}, nil
	} else if leftIsNum && !rightIsNum {
		// 80:service format
		return v1.PortBinding{}, fmt.Errorf("invalidate port binding [%s:%s] can not be number:string format", left, right)
	}
	// example.com:name
	return v1.PortBinding{
		ServiceName:       left,
		TargetServiceName: right,
		Protocol:          v1.ProtocolHTTP,
	}, nil
}

func ParsePorts(publish bool, args []string) (result []v1.PortBinding, _ error) {
	for _, arg := range args {
		var (
			binding v1.PortBinding
			err     error
		)

		arg, proto, _ := strings.Cut(arg, "/")
		parts := strings.Split(arg, ":")

		switch len(parts) {
		case 1:
			binding.TargetPort, err = parseSingle(parts[0])
			if err != nil {
				return nil, err
			}
		case 2:
			binding, err = parseTuple(parts[0], parts[1])
			if err != nil {
				return nil, err
			}
		case 3:
			binding, err = parseTriplet(parts[0], parts[1], parts[2])
			if err != nil {
				return nil, err
			}
		case 4:
			binding, err = parseQuad(!publish, parts[0], parts[1], parts[2], parts[3])
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("invalid syntax [%s] too many colon separated parts", arg)
		}

		if p, ok := validProto(proto); !ok {
			return nil, fmt.Errorf("invalid protocol [%s]", p)
		} else if binding.Protocol != "" && p != "" && binding.Protocol != p {
			return nil, fmt.Errorf("inferred protocol [%s] does not match requested protocol [%s]", binding.Protocol, p)
		} else if binding.Protocol == "" {
			binding.Protocol = p
		}

		binding.Publish = publish
		binding.Expose = !publish

		result = append(result, binding)
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

type Options struct {
	Name         string
	GenerateName string
	Namespace    string
	Annotations  map[string]string
	Labels       map[string]string
	PublishMode  v1.PublishMode
	Volumes      []v1.VolumeBinding
	Secrets      []v1.SecretBinding
	Services     []v1.ServiceBinding
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
			Ports:       opts.Ports,
			Image:       image,
			Volumes:     opts.Volumes,
			Secrets:     opts.Secrets,
			Services:    opts.Services,
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
