package appdefinition

import (
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/pullsecret"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type PullSecrets struct {
	objects  []kclient.Object
	keychain authn.Keychain
	app      *v1.AppInstance
	errs     []error
}

func NewPullSecrets(req router.Request, appInstance *v1.AppInstance) (*PullSecrets, error) {
	keychain, err := pullsecret.Keychain(req.Ctx, req.Client, appInstance.Namespace)
	if err != nil {
		return nil, err
	}

	return &PullSecrets{
		keychain: keychain,
		app:      appInstance,
	}, nil
}

func (p *PullSecrets) Err() error {
	if p == nil {
		return nil
	}
	return merr.NewErrors(p.errs...)
}

func (p *PullSecrets) Objects() []kclient.Object {
	if p == nil {
		return nil
	}
	return p.objects
}

func (p *PullSecrets) ForAcorn(acornName, image string) []corev1.LocalObjectReference {
	return p.ForContainer(acornName, []corev1.Container{
		{
			Image: image,
		},
	})
}

func (p *PullSecrets) ForContainer(containerName string, containers []corev1.Container) []corev1.LocalObjectReference {
	if p == nil {
		return nil
	}

	var images []string

	for _, container := range containers {
		images = append(images, container.Image)
	}

	secretName := name.SafeConcatName(containerName, "pull", p.app.ShortID())
	secret, err := pullsecret.ForImages(secretName, p.app.Status.Namespace, p.keychain, images...)
	if err != nil {
		p.errs = append(p.errs, err)
		return nil
	}

	p.objects = append(p.objects, secret)
	return []corev1.LocalObjectReference{
		{
			Name: secretName,
		},
	}
}
