package cryptokeys

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/imagesystem"
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
	client kclient.Client
	reveal bool
}

func (t *Translator) FromPublicName(ctx context.Context, namespace, name string) (string, string, error) {
	return namespace, strings.ReplaceAll(imagesystem.NormalizeServerAddress(name), ":", "-"), nil
}

func (t *Translator) ListOpts(ctx context.Context, namespace string, opts storage.ListOptions) (string, storage.ListOptions, error) {
	if opts.Predicate.Label == nil {
		opts.Predicate.Label = klabels.Everything()
	}
	reqs, _ := klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged:   "true",
		labels.AcornCryptoKey: "true",
	}).Requirements()
	opts.Predicate.Label = opts.Predicate.Label.Add(reqs...)
	return namespace, opts, nil
}

func (t *Translator) ToPublic(ctx context.Context, objs ...runtime.Object) (result []types.Object, _ error) {
	for _, obj := range objs {
		secret := obj.(*corev1.Secret)
		if secret.Type != apiv1.SecretTypeCryptoKey {
			continue
		}
		ck := &apiv1.CryptoKey{
			ObjectMeta: secret.ObjectMeta,
			Key:        string(secret.Data["key"]), // TODO: from fingerprint
		}
		ck.UID = ck.UID + "-s"
		//ck.Name = fingerprint // TODO: from fingerprint
		ck.OwnerReferences = nil
		ck.ManagedFields = nil
		if t.reveal {
			pass := string(secret.Data["key"])
			ck.Key = pass
		}
		result = append(result, ck)
	}

	return
}

func (t *Translator) FromPublic(ctx context.Context, obj runtime.Object) (types.Object, error) {
	input := obj.(*apiv1.Credential)

	secret := &corev1.Secret{
		ObjectMeta: input.ObjectMeta,
		Type:       corev1.SecretType(apiv1.SecretTypeCredential),
		Data: map[string][]byte{
			"serverAddress": []byte(input.ServerAddress),
			"username":      []byte(input.Username),
		},
	}

	if input.Password == nil {
		existing := &corev1.Secret{}
		existingNamespace, existingName, err := t.FromPublicName(ctx, input.Namespace, input.Name)
		if err != nil {
			return nil, err
		}

		err = t.client.Get(ctx, kclient.ObjectKey{Namespace: existingNamespace, Name: existingName}, existing)
		if err == nil {
			secret.Data["password"] = existing.Data["password"]
		} else if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		secret.Data["password"] = []byte(*input.Password)
	}

	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	secret.UID = ktypes.UID(strings.TrimSuffix(string(secret.UID), "-s"))
	secret.Labels[labels.AcornManaged] = "true"
	secret.Labels[labels.AcornCredential] = "true"
	secret.Name = strings.ReplaceAll(input.ServerAddress, ":", "-")

	return secret, nil
}

func (t *Translator) NewPublicList() types.ObjectList {
	return &apiv1.CredentialList{}
}

func (t *Translator) NewPublic() types.Object {
	return &apiv1.Credential{}
}
