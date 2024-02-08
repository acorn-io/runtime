package credentials

import (
	"context"
	"strings"

	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/labels"
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

func (t *Translator) FromPublicName(_ context.Context, namespace, name string) (string, string, error) {
	return namespace, strings.ReplaceAll(imagesystem.NormalizeServerAddress(name), ":", "-"), nil
}

func (t *Translator) ListOpts(_ context.Context, namespace string, opts storage.ListOptions) (string, storage.ListOptions, error) {
	if opts.Predicate.Label == nil {
		opts.Predicate.Label = klabels.Everything()
	}
	reqs, _ := klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged:    "true",
		labels.AcornCredential: "true",
	}).Requirements()
	opts.Predicate.Label = opts.Predicate.Label.Add(reqs...)
	return namespace, opts, nil
}

func (t *Translator) ToPublic(_ context.Context, objs ...runtime.Object) (result []types.Object, _ error) {
	for _, obj := range objs {
		secret := obj.(*corev1.Secret)
		if secret.Type != apiv1.SecretTypeCredential {
			continue
		}
		cred := &apiv1.Credential{
			ObjectMeta:    secret.ObjectMeta,
			ServerAddress: string(secret.Data["serverAddress"]),
			Username:      string(secret.Data["username"]),
		}
		cred.UID = cred.UID + "-s"
		cred.Name = cred.ServerAddress
		cred.OwnerReferences = nil
		cred.ManagedFields = nil
		if t.reveal {
			pass := string(secret.Data["password"])
			cred.Password = &pass
		}
		result = append(result, cred)
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
