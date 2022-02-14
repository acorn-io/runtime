package appdefinition

import (
	"encoding/json"

	cue_mod "github.com/ibuildthecloud/herd/cue.mod"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/cue"
	"github.com/ibuildthecloud/herd/schema"
)

const (
	HerdCueFile        = "herd.cue"
	ImageDataFile      = "images.json"
	BuildTransform     = "github.com/ibuildthecloud/herd/schema/v1/transform/build"
	NormalizeTransform = "github.com/ibuildthecloud/herd/schema/v1/transform/normalize"
)

type AppDefinition struct {
	ctx *cue.Context
}

func (a *AppDefinition) WithImageData(imageData v1.ImageData) (*AppDefinition, error) {
	imageDataBytes, err := json.Marshal(imageData)
	if err != nil {
		return nil, err
	}

	// Adding the ".cue" extension makes the cue parser merge the file. There's probably a better way to do that.
	return &AppDefinition{
		ctx: a.ctx.WithFile(ImageDataFile+".cue", imageDataBytes),
	}, nil
}

func FromAppImage(appImage *v1.AppImage) (*AppDefinition, error) {
	appDef, err := NewAppDefinition([]byte(appImage.Herdfile))
	if err != nil {
		return nil, err
	}

	return appDef.WithImageData(appImage.ImageData)
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

func (a *AppDefinition) AppSpec() (*v1.AppSpec, error) {
	v, err := a.ctx.Transform(NormalizeTransform)
	if err != nil {
		return nil, err
	}
	spec := &v1.AppSpec{}
	return spec, v.Decode(spec)
}

func (a *AppDefinition) BuildSpec() (*v1.BuildSpec, error) {
	v, err := a.ctx.Transform(BuildTransform)
	if err != nil {
		return nil, err
	}
	spec := &v1.BuildSpec{}
	return spec, v.Decode(spec)
}
