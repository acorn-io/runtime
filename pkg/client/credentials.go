package client

import (
	"context"
	"sort"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *client) CredentialCreate(ctx context.Context, serverAddress, username, password string, skipValidate bool) (*apiv1.Credential, error) {
	credential := &apiv1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverAddress,
			Namespace: c.Namespace,
		},
		ServerAddress: serverAddress,
		Username:      username,
		Password:      &password,
		SkipValidate:  skipValidate,
	}
	return credential, c.Client.Create(ctx, credential)
}

func (c *client) CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	credential := &apiv1.Credential{}
	return credential, c.Client.Get(ctx, kclient.ObjectKey{
		Name:      serverAddress,
		Namespace: c.Namespace,
	}, credential)
}

func (c *client) CredentialUpdate(ctx context.Context, serverAddress, username, password string, skipValidate bool) (*apiv1.Credential, error) {
	credential := &apiv1.Credential{}
	err := c.Client.Get(ctx, kclient.ObjectKey{
		Name:      serverAddress,
		Namespace: c.Namespace,
	}, credential)
	if err != nil {
		return nil, err
	}

	credential.Username = username
	credential.Password = &password
	credential.SkipValidate = skipValidate
	return credential, c.Client.Update(ctx, credential)
}

func (c *client) CredentialList(ctx context.Context) ([]apiv1.Credential, error) {
	result := &apiv1.CredentialList{}
	err := c.Client.List(ctx, result, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(result.Items, func(i, j int) bool {
		if result.Items[i].CreationTimestamp.Time == result.Items[j].CreationTimestamp.Time {
			return result.Items[i].Name < result.Items[j].Name
		}
		return result.Items[i].CreationTimestamp.After(result.Items[j].CreationTimestamp.Time)
	})

	return result.Items, nil
}

func (c *client) CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	credential, err := c.CredentialGet(ctx, serverAddress)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	err = c.Client.Delete(ctx, &apiv1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credential.Name,
			Namespace: credential.Namespace,
		},
	})
	if apierrors.IsNotFound(err) {
		return credential, nil
	}
	return credential, err
}
