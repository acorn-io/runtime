package appdefinition

import (
	cue_mod "github.com/ibuildthecloud/herd/cue.mod"
	v1 "github.com/ibuildthecloud/herd/pkg/api/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/cue"
	"github.com/ibuildthecloud/herd/schema"
)

const (
	HerdCueFile    = "herd.cue"
	ImageDataFile  = "images.json"
	BuildTransform = "github.com/ibuildthecloud/herd/schema/v1/transform/build"
)

type AppDefinition struct {
	ctx *cue.Context
}

func NewAppDefinition(data []byte) (*AppDefinition, error) {
	files := []cue.File{
		{
			Name: HerdCueFile,
			Data: data,
		},
	}
	ctx := cue.NewContext().
		WithNestedFS("schema", schema.Files).
		WithNestedFS("cue.mod", cue_mod.Files)
	ctx = ctx.WithFiles(files...)
	_, err := ctx.Value()
	return &AppDefinition{
		ctx: ctx,
	}, err
}

func (a *AppDefinition) BuildSpec() (*v1.BuildSpec, error) {
	v, err := a.ctx.Transform(BuildTransform)
	if err != nil {
		return nil, err
	}
	spec := &v1.BuildSpec{}
	return spec, v.Decode(spec)
}
