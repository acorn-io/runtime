package appdefinition

import (
	"path/filepath"
	"sort"

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
	Schema             = "github.com/ibuildthecloud/herd/schema/v1"
	AppType            = "#App"
)

type AppDefinition struct {
	ctx       *cue.Context
	imageData v1.ImagesData
}

func (a *AppDefinition) WithImageData(imageData v1.ImagesData) *AppDefinition {
	return &AppDefinition{
		ctx:       a.ctx,
		imageData: imageData,
	}
}

func FromAppImage(appImage *v1.AppImage) (*AppDefinition, error) {
	appDef, err := NewAppDefinition([]byte(appImage.Herdfile))
	if err != nil {
		return nil, err
	}

	return appDef.WithImageData(appImage.ImageData), nil
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
	err := ctx.Validate(Schema, AppType)
	return &AppDefinition{
		ctx: ctx,
	}, err
}

func (a *AppDefinition) AppSpec() (*v1.AppSpec, error) {
	app, err := a.ctx.Value()
	if err != nil {
		return nil, err
	}

	v, err := a.ctx.Encode(map[string]interface{}{
		"app":       app,
		"imageData": a.imageData,
	})
	if err != nil {
		return nil, err
	}

	v, err = a.ctx.TransformValue(v, NormalizeTransform)
	if err != nil {
		return nil, err
	}

	spec := &v1.AppSpec{}
	return spec, a.ctx.Decode(v, spec)
}

func addContainerFiles(fileSet map[string]bool, builds map[string]v1.ContainerImageBuilderSpec, cwd string) {
	for _, build := range builds {
		addContainerFiles(fileSet, build.Sidecars, cwd)
		if build.Build == nil {
			continue
		}
		fileSet[filepath.Join(cwd, build.Build.Dockerfile)] = true
	}
}

func addFiles(fileSet map[string]bool, builds map[string]v1.ImageBuilderSpec, cwd string) {
	for _, build := range builds {
		if build.Build == nil {
			continue
		}
		fileSet[filepath.Join(cwd, build.Build.Dockerfile)] = true
	}
}

func (a *AppDefinition) WatchFiles(cwd string) (result []string, _ error) {
	fileSet := map[string]bool{}
	spec, err := a.BuilderSpec()
	if err != nil {
		return nil, err
	}

	addContainerFiles(fileSet, spec.Containers, cwd)
	addFiles(fileSet, spec.Images, cwd)

	for k := range fileSet {
		result = append(result, k)
	}
	sort.Strings(result)
	return result, nil
}

func (a *AppDefinition) BuilderSpec() (*v1.BuilderSpec, error) {
	v, err := a.ctx.Transform(BuildTransform)
	if err != nil {
		return nil, err
	}
	spec := &v1.BuilderSpec{}
	return spec, a.ctx.Decode(v, spec)
}
