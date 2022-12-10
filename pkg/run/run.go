package run

import (
	"context"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/namespace"
	"github.com/acorn-io/namegenerator"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

func Run(ctx context.Context, c client.Client, app *v1.AppInstance) (*v1.AppInstance, error) {
	if err := namespace.Ensure(ctx, c, app.Namespace); err != nil {
		return nil, err
	}

	if app.Name == "" && app.GenerateName == "" {
		app.Name = NameGenerator.Generate()
	}

	if app.Labels == nil {
		app.Labels = map[string]string{}
	}

	return app, c.Create(ctx, app)
}
