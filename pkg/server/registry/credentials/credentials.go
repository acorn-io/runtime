package credentials

import (
	"context"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/acorn/pkg/watcher"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) *Storage {
	return &Storage{
		TableConvertor: tables.CredentialConverter,
		client:         c,
	}
}

type Storage struct {
	rest.TableConvertor

	client client.WithWatch
}

func (s *Storage) NewList() runtime.Object {
	return &apiv1.CredentialList{}
}

func (s *Storage) NamespaceScoped() bool {
	return true
}

func (s *Storage) New() runtime.Object {
	return &apiv1.Credential{}
}

func secretToCredential(secret corev1.Secret) *apiv1.Credential {
	cred := &apiv1.Credential{
		ObjectMeta:    secret.ObjectMeta,
		ServerAddress: string(secret.Data["serverAddress"]),
		Storage:       apiv1.CredentialStorageTypeCluster,
		Username:      string(secret.Data["username"]),
	}
	cred.UID = cred.UID + "-s"
	cred.Name = cred.ServerAddress
	return cred
}

func toSecretName(credName string) string {
	return strings.ReplaceAll(credName, ":", "-")
}

func credToSecret(input *apiv1.Credential) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: input.ObjectMeta,
		Type:       corev1.SecretType(apiv1.SecretTypeCredential),
		StringData: map[string]string{
			"serverAddress": input.ServerAddress,
			"username":      input.Username,
			"password":      input.Password,
		},
	}

	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	secret.UID = types.UID(strings.TrimSuffix(string(secret.UID), "-s"))
	secret.Labels[labels.AcornManaged] = "true"
	secret.Labels[labels.AcornCredential] = "true"
	secret.Name = strings.ReplaceAll(input.ServerAddress, ":", "-")

	return secret
}

func (s *Storage) isUnique(ctx context.Context, serverAddress string) error {
	credsObj, err := s.List(ctx, nil)
	if err != nil {
		return err
	}

	creds := credsObj.(*apiv1.CredentialList)
	for _, cred := range creds.Items {
		if cred.ServerAddress == serverAddress {
			return apierror.NewAlreadyExists(schema.GroupResource{
				Group:    api.Group,
				Resource: "credentials",
			}, cred.ServerAddress)
		}
	}

	return nil
}

func normalizeDockerIO(s string) string {
	if s == "docker.io" {
		return "index.docker.io"
	}
	return s
}

func (s *Storage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	input := obj.(*apiv1.Credential)
	input.ServerAddress = normalizeDockerIO(input.ServerAddress)

	if err := s.isUnique(ctx, input.ServerAddress); err != nil {
		return nil, err
	}

	secret := credToSecret(input)

	if err := s.client.Create(ctx, secret); err != nil {
		return nil, err
	}

	return secretToCredential(*secret), nil
}

func (s *Storage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	name = normalizeDockerIO(name)

	obj, err := s.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}
	cred := obj.(*apiv1.Credential)
	if deleteValidation != nil {
		if err := deleteValidation(ctx, obj); err != nil {
			return nil, false, err
		}
	}
	return obj, true, s.client.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      toSecretName(cred.Name),
			Namespace: cred.Namespace,
		},
	})
}

func (s *Storage) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	name = normalizeDockerIO(name)

	oldCred, err := s.CredGet(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}

	newObj, err := objInfo.UpdatedObject(ctx, oldCred)
	if err != nil {
		return nil, false, err
	}

	if updateValidation != nil {
		if err := updateValidation(ctx, newObj, oldCred); err != nil {
			return nil, false, err
		}
	}

	newCred := newObj.(*apiv1.Credential)

	if oldCred.ServerAddress != newCred.ServerAddress {
		if err := s.isUnique(ctx, newCred.ServerAddress); err != nil {
			return nil, false, err
		}
	}

	secret := credToSecret(newCred)
	if err := s.client.Update(ctx, secret); err != nil {
		return nil, false, err
	}

	return secretToCredential(*secret), true, nil
}

func (s *Storage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return s.CredGet(ctx, name, options)
}

func (s *Storage) CredGet(ctx context.Context, name string, options *metav1.GetOptions) (*apiv1.Credential, error) {
	name = normalizeDockerIO(name)

	credsObj, err := s.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	creds := credsObj.(*apiv1.CredentialList)
	for _, cred := range creds.Items {
		if cred.Name == name {
			return &cred, nil
		}
	}

	return nil, apierror.NewNotFound(schema.GroupResource{
		Group:    api.Group,
		Resource: "credentials",
	}, name)
}

func (s *Storage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	ns, _ := request.NamespaceFrom(ctx)
	secrets := &corev1.SecretList{}
	err := s.client.List(ctx, secrets, &client.ListOptions{
		Namespace: ns,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornCredential: "true",
			labels.AcornManaged:    "true",
		}),
	})
	if err != nil {
		return nil, err
	}

	result := &apiv1.CredentialList{
		ListMeta: secrets.ListMeta,
	}

	for _, secret := range secrets.Items {
		result.Items = append(result.Items, *secretToCredential(secret))
	}

	return result, nil
}

func (s *Storage) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	ns, _ := request.NamespaceFrom(ctx)

	opts := watcher.ListOptions(ns, options)
	opts.FieldSelector = nil
	opts.Raw.FieldSelector = ""
	opts.LabelSelector = klabels.SelectorFromSet(map[string]string{
		labels.AcornCredential: "true",
		labels.AcornManaged:    "true",
	})

	w, err := s.client.Watch(ctx, &corev1.SecretList{}, opts)
	if err != nil {
		return nil, err
	}

	return watcher.Transform(w, func(object runtime.Object) []runtime.Object {
		sec := object.(*corev1.Secret)
		return []runtime.Object{
			secretToCredential(*sec),
		}
	}), nil
}
