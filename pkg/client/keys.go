package client

import (
	"context"
	"crypto"
	"sort"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	acornsign "github.com/acorn-io/acorn/pkg/cosign"
	"github.com/acorn-io/baaah/pkg/name"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) KeyCreate(ctx context.Context, key crypto.PublicKey) (*apiv1.PublicKey, error) {
	pem, fingerprint, err := acornsign.PemEncodeCryptoPublicKey(key)
	if err != nil {
		return nil, err
	}

	pk := &apiv1.PublicKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.SafeConcatName("pub", fingerprint),
			Namespace: c.Namespace,
		},
		Key:         string(pem),
		Fingerprint: fingerprint,
	}

	return pk, c.Client.Create(ctx, pk)
}

func (c *DefaultClient) KeyGet(ctx context.Context, name string) (*apiv1.PublicKey, error) {
	pubkey := &apiv1.PublicKey{}
	return pubkey, c.Client.Get(ctx, kclient.ObjectKey{
		Namespace: c.Namespace,
		Name:      name,
	}, pubkey)
}

func (c *DefaultClient) KeyList(ctx context.Context) ([]apiv1.PublicKey, error) {
	result := &apiv1.PublicKeyList{}
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

func (c *DefaultClient) KeyDelete(ctx context.Context, name string) (*apiv1.PublicKey, error) {
	pubkey, err := c.KeyGet(ctx, name)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	err = c.Client.Delete(ctx, pubkey)
	if apierrors.IsNotFound(err) {
		return pubkey, nil
	}
	return pubkey, err
}
