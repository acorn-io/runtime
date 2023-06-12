package secrets

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/digest"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ref"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/acorn/pkg/volume"
	"github.com/acorn-io/aml"
	"github.com/acorn-io/aml/pkg/replace"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"github.com/rancher/wrangler/pkg/merr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	serviceTokens = sets.NewString("address",
		"hostname",
		"port",
		"ports",
		"data",
		"secrets",
		"secret",
		"endpoint",
		"host",
		"hostname")
)

type ErrInterpolation struct {
	v1.ExpressionError
}

func (e *ErrInterpolation) Error() string {
	return e.ExpressionError.String()
}

type Interpolator struct {
	secretName string

	app           *v1.AppInstance
	data          map[string][]byte
	incomplete    map[string]bool
	client        kclient.Client
	ctx           context.Context
	errs          *[]error
	namespace     string
	containerName string
	jobName       string
}

func NewInterpolator(ctx context.Context, c kclient.Client, app *v1.AppInstance) *Interpolator {
	return &Interpolator{
		secretName: "secrets-" + app.ShortID(),
		app:        app,
		data:       map[string][]byte{},
		incomplete: map[string]bool{},
		client:     c,
		ctx:        ctx,
		namespace:  app.Status.Namespace,
		errs:       &[]error{},
	}
}

func (i *Interpolator) Err() error {
	return merr.NewErrors(*i.errs...)
}

func (i *Interpolator) ForJob(jobName string) *Interpolator {
	cp := *i
	cp.jobName = jobName
	cp.containerName = ""
	return &cp
}

func (i *Interpolator) ForContainer(containerName string) *Interpolator {
	cp := *i
	cp.jobName = ""
	cp.containerName = containerName
	return &cp
}

func (i *Interpolator) Incomplete() bool {
	if i.jobName != "" {
		return i.incomplete[i.jobName]
	} else if i.containerName != "" {
		return i.incomplete[i.containerName]
	}
	return len(i.incomplete) > 0
}

func (i *Interpolator) AddMissingAnnotations(annotations map[string]string) {
	if i.Incomplete() {
		annotations[apply.AnnotationUpdate] = "false"
		annotations[apply.AnnotationCreate] = "false"
	}
}

func (i *Interpolator) addContent(newValue string) string {
	dataKey := digest.SHA256(newValue)
	i.data[dataKey] = []byte(newValue)
	return dataKey
}

func (i *Interpolator) SecretName() string {
	return i.secretName
}

func (i *Interpolator) ToVolumeMount(filename string, file v1.File) corev1.VolumeMount {
	data, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		i.saveError(err)
		return corev1.VolumeMount{}
	}

	newValue, err := i.replace(string(data))
	if err != nil {
		i.saveError(err)
		return corev1.VolumeMount{}
	}

	suffix := ""
	if volume.NormalizeMode(file.Mode) != "" {
		suffix = "-" + file.Mode
	}

	return corev1.VolumeMount{
		Name:      i.secretName + suffix,
		MountPath: path.Join("/", filename),
		SubPath:   i.addContent(newValue),
	}
}

func (i *Interpolator) resolveApp(keyName string) (string, bool, error) {
	switch keyName {
	case "name":
		return i.app.Name, true, nil
	case "project":
		project := i.app.Labels[labels.AcornProjectName]
		if project != "" {
			return project, true, nil
		}
		return i.app.Namespace, true, nil
	case "namespace":
		return i.app.Status.Namespace, true, nil
	case "image":
		if tags.IsLocalReference(i.app.Status.AppImage.ID) {
			return i.app.Status.AppImage.ID, true, nil
		} else if i.app.Status.AppImage.ID != "" && i.app.Status.AppImage.Digest != "" {
			tag, err := name.NewTag(i.app.Status.AppImage.ID)
			if err != nil {
				return "", false, err
			}
			return tag.Digest(i.app.Status.AppImage.Digest).String(), true, nil
		}
	case "imageName":
		return i.app.Status.AppImage.Name, true, nil
	case "commitSha":
		return i.app.Status.AppImage.VCS.Revision, true, nil
	}
	return "", false, nil
}

func (i *Interpolator) resolveSecrets(secretName []string, keyName string) (string, bool, error) {
	secret := &corev1.Secret{}
	ns := i.namespace
	secretNames := secretName

	// support interpolation of external secrets
	if len(secretName) > 0 && strings.HasPrefix(secretName[0], "external:") {
		secretNames = append([]string{
			strings.TrimPrefix(secretName[0], "external:"),
		}, secretName[1:]...)
		ns = i.app.Namespace
	}

	err := ref.Lookup(i.ctx, i.client, secret, ns, secretNames...)
	if apierrors.IsNotFound(err) {
		return "", false, &ErrInterpolation{
			ExpressionError: v1.ExpressionError{
				Error: notFound("secret", secretName...),
			},
		}
	} else if err != nil {
		return "", false, err
	}
	return string(secret.Data[keyName]), true, nil
}

func notFound(kind string, keys ...string) string {
	return kind + " [" + strings.Join(keys, ".") + "] not found"
}

func splitServiceProperty(parts []string) (head []string, tail []string, err error) {
	for i, part := range parts {
		if serviceTokens.Has(part) {
			return parts[:i], parts[i:], nil
		}
	}
	return nil, nil, fmt.Errorf("service lookup [%s] must include one service propery [%s]",
		strings.Join(parts, "."), strings.Join(serviceTokens.List(), ","))
}

func (i *Interpolator) serviceProperty(svc *v1.ServiceInstance, prop string, extra []string) (resolved string, err error) {
	// sanity check that our serviceToken map is complete, because this will fail if you add
	// a new case but don't add to the serviceToken set then it won't get to the switch
	if !serviceTokens.Has(prop) {
		return "", fmt.Errorf("invalid property [%s] to lookup on service [%s]", prop, svc.Name)
	}

	defer func() {
		if err != nil {
			return
		}
		resolved, err = i.resolveNestedData(svc, resolved)
	}()

	switch prop {
	case "secret", "secrets":
		if len(extra) != 2 {
			return "", fmt.Errorf("invalid secret lookup on service [%s] key must be at least two parts, got %v", svc.Name, extra)
		}
		secret := &corev1.Secret{}
		err := ref.Lookup(i.ctx, i.client, secret, svc.Namespace, extra[0])
		if apierrors.IsNotFound(err) {
			return "", &ErrInterpolation{
				ExpressionError: v1.ExpressionError{
					Error: notFound("secret", extra[0]),
				},
			}
		} else if err != nil {
			return "", err
		}
		return string(secret.Data[extra[1]]), nil
	case "endpoint":
		if len(svc.Status.Endpoints) > 0 {
			return svc.Status.Endpoints[0].Address, nil
		}
		return "", &ErrInterpolation{
			ExpressionError: v1.ExpressionError{
				Error: fmt.Sprintf("endpoint on service [%s] undefined", svc.Name),
			},
		}
	case "address", "host", "hostname":
		if svc.Spec.Address != "" {
			return svc.Spec.Address, nil
		}
		cfg, err := config.Get(i.ctx, i.client)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s.%s", svc.Name, svc.Namespace, cfg.InternalClusterDomain), nil
	case "port", "ports":
		if len(extra) != 1 {
			return "", fmt.Errorf("can not lookup ports expecting single number, got [%s]", strings.Join(extra, "."))
		}
		for _, port := range svc.Spec.Ports {
			p := port.Complete()
			if strconv.Itoa(int(p.Port)) == extra[0] {
				return strconv.Itoa(int(p.TargetPort)), nil
			}
		}
		return "", fmt.Errorf("failed to find port [%s] defined on service [%s]", extra[0], svc.Name)
	case "data":
		expr := "@{" + strings.Join(extra, ".") + "}"
		v, err := aml.Interpolate(svc.Spec.Data, expr)
		return v, err
	default:
		return "", fmt.Errorf("invalid property [%s] to lookup on service [%s]", prop, svc.Name)
	}
}

func (i *Interpolator) resolveNestedData(svc *v1.ServiceInstance, val string) (string, error) {
	if !strings.Contains(val, "@{") {
		return val, nil
	}

	var app v1.AppInstance
	if err := i.client.Get(i.ctx, router.Key(svc.Labels[labels.AcornAppNamespace], svc.Labels[labels.AcornAppName]), &app); err != nil {
		return "", err
	}

	val, _, err := NewInterpolator(i.ctx, i.client, &app).resolve(val)
	return val, err
}

func (i *Interpolator) resolveServices(parts []string) (string, bool, error) {
	serviceName, properties, err := splitServiceProperty(parts)
	if err != nil {
		return "", false, err
	}

	svc := &v1.ServiceInstance{}
	err = ref.Lookup(i.ctx, i.client, svc, i.namespace, serviceName...)
	if apierrors.IsNotFound(err) {
		return "", false, &ErrInterpolation{
			ExpressionError: v1.ExpressionError{
				Error: notFound("service", serviceName...),
			},
		}
	}
	if err != nil {
		return "", false, err
	}

	ret, err := i.serviceProperty(svc, properties[0], properties[1:])
	return ret, true, err
}

func (i *Interpolator) resolve(token string) (_ string, _ bool, err error) {
	defer func() {
		if target := (*ErrInterpolation)(nil); errors.As(err, &target) {
			target.Expression = token
			err = target
		}
	}()

	scheme, tail, ok := strings.Cut(token, "://")
	if ok {
		switch scheme {
		case "secret", "secrets":
			parts := strings.Split(tail, "/")
			if len(parts) == 2 {
				return i.resolveSecrets([]string{parts[0]}, parts[1])
			}
		}
	}

	parts := strings.Split(strings.TrimSpace(token), ".")
	switch parts[0] {
	case "service", "services":
		if len(parts) < 3 {
			return "", false, fmt.Errorf("invalid expression [%s], must have at least three parts separated by \".\"", token)
		}
		return i.resolveServices(parts[1:])
	case "secret", "secrets":
		if len(parts) < 3 {
			return "", false, fmt.Errorf("invalid expression [%s], must have at least three parts separated by \".\"", token)
		}
		return i.resolveSecrets(parts[1:len(parts)-1], parts[len(parts)-1])
	case "acorn", "app":
		if len(parts) != 2 {
			return "", false, fmt.Errorf("invalid expression [%s], must have two parts separated by \".\"", token)
		}
		return i.resolveApp(parts[1])
	case "image", "images":
		if len(parts) != 2 {
			return "", false, fmt.Errorf("invalid expression [%s], must have two parts separated by \".\"", token)
		}
		return i.resolveImages(parts[1])
	default:
		return "", false, nil
	}
}

func (i *Interpolator) resolveImages(imageName string) (string, bool, error) {
	img, ok := i.app.Status.AppSpec.Images[imageName]
	if !ok {
		return "", false, nil
	}
	result := strings.TrimPrefix(img.Image, "sha256:")
	return result, result != "", nil
}

func (i *Interpolator) replace(content string) (string, error) {
	content, err := replace.Replace(content, "@{", "}", i.resolve)
	if err != nil {
		return "", err
	}
	return replace.Replace(content, nacl.EncPrefix, nacl.EncSuffix, func(s string) (string, bool, error) {
		data, err := nacl.DecryptNamespacedData(i.ctx, i.client, []byte(nacl.EncPrefix+s+nacl.EncSuffix), i.app.Namespace)
		return string(data), true, err
	})
}

func (i *Interpolator) saveError(err error) {
	var exprError v1.ExpressionError
	if e := (*ErrInterpolation)(nil); errors.As(err, &e) {
		exprError = e.ExpressionError
	} else {
		exprError = v1.ExpressionError{
			Error: err.Error(),
		}
	}
	if i.containerName != "" {
		i.incomplete[i.containerName] = true
		c := i.app.Status.AppStatus.Containers[i.containerName]
		c.ExpressionErrors = append(c.ExpressionErrors, exprError)
		i.app.Status.AppStatus.Containers[i.containerName] = c
	} else if i.jobName != "" {
		i.incomplete[i.jobName] = true
		c := i.app.Status.AppStatus.Jobs[i.jobName]
		c.ExpressionErrors = append(c.ExpressionErrors, exprError)
		i.app.Status.AppStatus.Jobs[i.jobName] = c
	} else {
		i.incomplete[""] = true
		*i.errs = append(*i.errs, fmt.Errorf("unqualified expression error: %w", err))
	}
}

func (i *Interpolator) ToEnv(key, value string) corev1.EnvVar {
	newValue, err := i.replace(value)
	if err != nil {
		i.saveError(err)
		return corev1.EnvVar{
			Name:  key,
			Value: value,
		}
	}
	if value == newValue {
		return corev1.EnvVar{
			Name:  key,
			Value: value,
		}
	}

	return corev1.EnvVar{
		Name: key,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: i.secretName,
				},
				Key: i.addContent(newValue),
			},
		},
	}
}

func (i *Interpolator) Objects() []kclient.Object {
	if len(i.data) == 0 {
		return nil
	}
	return []kclient.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      i.secretName,
				Namespace: i.app.Status.Namespace,
				Labels:    labels.Managed(i.app),
			},
			Data: i.data,
		},
	}
}
