package run

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/namegenerator"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

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

func Run(ctx context.Context, c client.Client, app *v1.AppInstance) (*v1.AppInstance, error) {
	if err := createNamespace(ctx, c, app.Namespace); err != nil {
		return nil, err
	}

	if app.Name == "" && app.GenerateName == "" {
		app.Name = NameGenerator.Generate()
	}

	if app.Labels == nil {
		app.Labels = map[string]string{}
	}

	app.Labels[labels.AcornRootNamespace] = app.Namespace
	app.Labels[labels.AcornManaged] = "true"
	return app, c.Create(ctx, app)
}
