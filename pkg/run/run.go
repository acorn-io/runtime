package run

import (
	"context"
	"time"

	"github.com/acorn-io/namegenerator"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/namespace"
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
