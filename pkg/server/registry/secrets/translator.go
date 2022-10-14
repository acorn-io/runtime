package secrets

import (
	"context"
	"sort"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/mink/pkg/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Translator struct {
	c      kclient.Client
	expose bool
}

func (t *Translator) FromPublicName(ctx context.Context, namespace, name string) (string, string, error) {
	i := strings.LastIndex(name, ".")
	if i == -1 || i+1 > len(name) {
		return namespace, name, nil
	}

	prefix := name[:i]
	secretName := name[i+1:]

	apps := &v1.AppInstanceList{}
	err := t.c.List(ctx, apps, &kclient.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return "", "", err
	}

	for _, app := range apps.Items {
		if app.Name != prefix {
			continue
		}
		for _, binding := range app.Spec.Secrets {
			if binding.Target == secretName {
				return namespace, binding.Secret, nil
			}
		}
	}

	secrets := &corev1.SecretList{}
	err = t.c.List(ctx, secrets, &kclient.ListOptions{
		Namespace: namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornAppName:    prefix,
			labels.AcornSecretName: secretName,
		}),
	})
	if err != nil {
		return "", "", err
	}
	if len(secrets.Items) == 1 {
		return namespace, secrets.Items[0].Name, nil
	}

	return namespace, name, nil
}

func (t *Translator) ListOpts(namespace string, opts storage.ListOptions) (string, storage.ListOptions) {
	return namespace, opts
}

func (t *Translator) ToPublic(objs ...runtime.Object) (result []types.Object) {
	for _, obj := range objs {
		var keys []string
		secret := obj.(*corev1.Secret)
		if ignore(secret) {
			continue
		}
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
		if t.expose {
			sec.Data = secret.Data
		}
		result = append(result, sec)
	}
	return
}

func (t *Translator) FromPublic(ctx context.Context, obj runtime.Object) (types.Object, error) {
	secret := obj.(*apiv1.Secret)
	if secret.Data == nil {
		existingNamespace, existingName, err := t.FromPublicName(ctx, secret.Namespace, secret.Name)
		if err != nil {
			return nil, err
		}
		existing := &corev1.Secret{}
		err = t.c.Get(ctx, kclient.ObjectKey{Namespace: existingNamespace, Name: existingName}, existing)
		if err == nil {
			secret.Data = existing.Data
		} else if !apierrors.IsNotFound(err) {
			return nil, err
		}
	}

	newSecret := &corev1.Secret{
		ObjectMeta: secret.ObjectMeta,
		Data:       secret.Data,
		Type:       corev1.SecretType(secret.Type),
	}
	newSecret.UID = ktypes.UID(strings.TrimSuffix(string(newSecret.UID), "-s"))

	if newSecret.Type == "" {
		newSecret.Type = v1.SecretTypeOpaque
	} else {
		newSecret.Type = v1.SecretTypePrefix + newSecret.Type
	}
	return newSecret, nil
}

func (t *Translator) NewPublic() types.Object {
	return &apiv1.Secret{}
}

func (t *Translator) NewPublicList() types.ObjectList {
	return &apiv1.SecretList{}
}

func ignore(secret *corev1.Secret) bool {
	return !strings.HasPrefix(string(secret.Type), "secrets.acorn.io/")
}
