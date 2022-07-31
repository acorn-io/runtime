package secrets

import (
	"context"
	"sort"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/acorn/pkg/watcher"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ignore(secret *corev1.Secret) bool {
	return !strings.HasPrefix(string(secret.Type), "secrets.acorn.io/")
}

func NewStorage(c client.WithWatch) *Storage {
	return &Storage{
		TableConvertor: tables.SecretConverter,
		client:         c,
	}
}

type Storage struct {
	rest.TableConvertor

	client client.WithWatch
}

func (s *Storage) NewList() runtime.Object {
	return &apiv1.SecretList{}
}

func (s *Storage) NamespaceScoped() bool {
	return true
}

func (s *Storage) New() runtime.Object {
	return &apiv1.Secret{}
}

func coreSecretToSecret(secret *corev1.Secret) *apiv1.Secret {
	var keys []string
	for key := range secret.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	sec := &apiv1.Secret{
		ObjectMeta: secret.ObjectMeta,
		Type:       strings.TrimPrefix(string(secret.Type), v1.SecretTypePrefix),
		Keys:       keys,
	}
	sec.UID = sec.UID + "-s"
	return sec
}

func (s *Storage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	secret := obj.(*apiv1.Secret)
	newSecret := &corev1.Secret{
		ObjectMeta: secret.ObjectMeta,
		Data:       secret.Data,
		Type:       corev1.SecretType(secret.Type),
	}

	if newSecret.Type == "" {
		newSecret.Type = v1.SecretTypeOpaque
	} else {
		newSecret.Type = v1.SecretTypePrefix + newSecret.Type
	}

	if !v1.SecretTypes[newSecret.Type] {
		return nil, apierror.NewInvalid(schema.GroupKind{
			Group: api.Group,
			Kind:  "Secret",
		}, newSecret.Name, field.ErrorList{{
			Type:     "string",
			Field:    "type",
			BadValue: secret.Type,
			Detail:   "Invalid secret type",
		}})
	}

	err := s.client.Create(ctx, newSecret)
	if err != nil {
		return nil, err
	}

	return coreSecretToSecret(newSecret), nil
}

func (s *Storage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	ns, _ := request.NamespaceFrom(ctx)

	secrets := &corev1.SecretList{}
	err := s.client.List(ctx, secrets, &client.ListOptions{
		Namespace: ns,
	})
	if err != nil {
		return nil, err
	}

	result := &apiv1.SecretList{
		ListMeta: secrets.ListMeta,
	}
	for _, secret := range secrets.Items {
		if ignore(&secret) {
			continue
		}
		result.Items = append(result.Items, *coreSecretToSecret(&secret))
	}

	return result, nil
}

func (s *Storage) resolveName(ctx context.Context, ns, name string) (string, error) {
	i := strings.LastIndex(name, ".")
	if i == -1 || i+1 > len(name) {
		return name, nil
	}

	prefix := name[:i]
	secretName := name[i+1:]

	apps := &v1.AppInstanceList{}
	err := s.client.List(ctx, apps, &kclient.ListOptions{
		Namespace: ns,
	})
	if err != nil {
		return "", err
	}

	for _, app := range apps.Items {
		if app.Name != prefix {
			continue
		}
		for _, binding := range app.Spec.Secrets {
			if binding.Target == secretName {
				return binding.Secret, nil
			}
		}
	}

	secrets := &corev1.SecretList{}
	err = s.client.List(ctx, secrets, &kclient.ListOptions{
		Namespace: ns,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornRootPrefix: prefix,
			labels.AcornSecretName: secretName,
		}),
	})
	if err != nil {
		return "", err
	}
	if len(secrets.Items) == 1 {
		return secrets.Items[0].Name, nil
	}

	return name, nil
}

func (s *Storage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, _ := request.NamespaceFrom(ctx)
	name, err := s.resolveName(ctx, ns, name)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	err = s.client.Get(ctx, router.Key(ns, name), secret)
	if err != nil {
		return nil, err
	}

	if ignore(secret) {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "secrets",
		}, name)
	}

	return coreSecretToSecret(secret), nil
}

func (s *Storage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	secret, err := s.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}

	if deleteValidation != nil {
		if err := deleteValidation(ctx, secret); err != nil {
			return nil, false, err
		}
	}

	err = s.client.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.(*apiv1.Secret).Name,
			Namespace: secret.(*apiv1.Secret).Namespace,
		},
	})
	if err != nil {
		return nil, false, err
	}

	return secret, true, nil
}

func (s *Storage) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	secret, err := s.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}

	newSecret, err := objInfo.UpdatedObject(ctx, secret)
	if err != nil {
		return nil, false, err
	}

	if updateValidation != nil {
		if err := updateValidation(ctx, newSecret, secret); err != nil {
			return nil, false, err
		}
	}

	newV1Secret := newSecret.(*apiv1.Secret)

	newCoreSecret := &corev1.Secret{
		ObjectMeta: newV1Secret.ObjectMeta,
		Data:       newV1Secret.Data,
		Type:       corev1.SecretType(v1.SecretTypePrefix + secret.(*apiv1.Secret).Type),
	}
	newCoreSecret.UID = types.UID(strings.TrimSuffix(string(newCoreSecret.UID), "-s"))

	err = s.client.Update(ctx, newCoreSecret)
	if err != nil {
		return nil, false, err
	}

	return coreSecretToSecret(newCoreSecret), true, nil
}

func (s *Storage) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	ns, _ := request.NamespaceFrom(ctx)
	w, err := s.client.Watch(ctx, &corev1.SecretList{}, watcher.ListOptions(ns, options))
	if err != nil {
		return nil, err
	}
	return watcher.Transform(w, func(object runtime.Object) []runtime.Object {
		secret := object.(*corev1.Secret)
		if ignore(secret) {
			return nil
		}
		return []runtime.Object{
			coreSecretToSecret(secret),
		}
	}), nil
}
