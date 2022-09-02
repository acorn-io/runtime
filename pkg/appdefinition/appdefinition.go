package appdefinition

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	cue2 "cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	cue_mod "github.com/acorn-io/acorn/cue.mod"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/acorn-io/acorn/schema"
	"sigs.k8s.io/yaml"
)

const (
	AcornCueFile       = "Acornfile"
	ImageDataFile      = "images.json"
	BuildDataFile      = "build.json"
	BuildTransform     = "github.com/acorn-io/acorn/schema/v1/transform/build"
	NormalizeTransform = "github.com/acorn-io/acorn/schema/v1/transform/normalize"
	Schema             = "github.com/acorn-io/acorn/schema/v1"
	AppType            = "#App"
)

var Defaults = []byte(`

args: dev: bool | *false
profiles: dev: dev: bool | *true
`)

type AppDefinition struct {
	ctx        *cue.Context
	imageDatas []v1.ImagesData
}

func FromAppImage(appImage *v1.AppImage) (*AppDefinition, error) {
	appDef, err := NewAppDefinition([]byte(appImage.Acornfile))
	if err != nil {
		return nil, err
	}

	appDef = appDef.WithImageData(appImage.ImageData)
	return appDef, err
}

func (a *AppDefinition) WithImageData(imageData v1.ImagesData) *AppDefinition {
	return &AppDefinition{
		ctx:        a.ctx,
		imageDatas: append(a.imageDatas, imageData),
	}
}

func NewAppDefinition(data []byte) (*AppDefinition, error) {
	files := []cue.File{
		{
			Name: AcornCueFile + ".cue",
			Data: append(data, Defaults...),
			Parser: func(name string, src any) (*ast.File, error) {
				return parseFile(AcornCueFile, src)
			},
		},
	}
	ctx := cue.NewContext().
		WithNestedFS("schema", schema.Files).
		WithNestedFS("cue.mod", cue_mod.Files)
	ctx = ctx.WithFiles(files...)
	ctx = ctx.WithSchema(Schema, AppType)
	_, err := ctx.Value()
	if err != nil {
		return nil, err
	}
	appDef := &AppDefinition{
		ctx: ctx,
	}
	_, err = appDef.AppSpec()
	if err != nil {
		return nil, err
	}
	return appDef, nil
}

func assignAcornImage(originalImage string, build *v1.AcornBuild, image string) (string, *v1.AcornBuild) {
	if build == nil {
		build = &v1.AcornBuild{}
	}
	if build.OriginalImage == "" {
		build.OriginalImage = originalImage
	}
	return image, build
}

func assignImage(originalImage string, build *v1.Build, image string) (string, *v1.Build) {
	if build == nil {
		build = &v1.Build{}
	}
	if build.BaseImage == "" {
		build.BaseImage = originalImage
	}
	return image, build
}

func (a *AppDefinition) getArgsForProfile(args map[string]any, profiles []string) (map[string]any, error) {
	val, err := a.ctx.Value()
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		optional := false
		if strings.HasSuffix(profile, "?") {
			optional = true
			profile = profile[:len(profile)-1]
		}
		path := cue2.ParsePath(fmt.Sprintf("profiles.%s", profile))
		pValue := val.LookupPath(path)
		if !pValue.Exists() {
			if !optional {
				return nil, fmt.Errorf("failed to find profile %s", profile)
			}
			continue
		}

		if args == nil {
			args = map[string]any{}
		}

		inValue, err := a.ctx.Encode(args)
		if err != nil {
			return nil, err
		}

		newArgs := map[string]any{}
		err = pValue.Unify(*inValue).Decode(&newArgs)
		if err != nil {
			return nil, cue.WrapErr(err)
		}
		args = newArgs
	}

	return args, nil
}

func (a *AppDefinition) WithArgs(args map[string]any, profiles []string) (*AppDefinition, map[string]any, error) {
	args, err := a.getArgsForProfile(args, profiles)
	if err != nil {
		return nil, nil, err
	}
	if len(args) == 0 {
		return a, args, nil
	}
	data, err := json.Marshal(map[string]any{
		"args": args,
	})
	if err != nil {
		return nil, nil, err
	}
	return &AppDefinition{
		ctx:        a.ctx.WithFile("args.cue", data),
		imageDatas: a.imageDatas,
	}, args, nil
}

func (a *AppDefinition) YAML() (string, error) {
	jsonData, err := a.JSON()
	if err != nil {
		return "", err
	}
	data := map[string]any{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return "", err
	}
	y, err := yaml.Marshal(data)
	return string(y), err
}

func (a *AppDefinition) JSON() (string, error) {
	app, err := a.ctx.Value()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(app, "", "  ")
	return string(data), err
}

func (a *AppDefinition) AppSpec() (*v1.AppSpec, error) {
	app, err := a.ctx.Value()
	if err != nil {
		return nil, err
	}

	objs := map[string]any{}
	for _, key := range []string{"containers", "jobs", "acorns", "secrets", "volumes", "images", "labels", "annotations"} {
		v := app.LookupPath(cue2.ParsePath(key))
		if v.Exists() {
			objs[key] = v
		}
	}

	newApp, err := a.ctx.Encode(objs)
	if err != nil {
		return nil, err
	}

	spec := &v1.AppSpec{}
	if err := a.ctx.Decode(newApp, spec); err != nil {
		return nil, err
	}

	for _, imageData := range a.imageDatas {
		for c, con := range imageData.Containers {
			if conSpec, ok := spec.Containers[c]; ok {
				conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, con.Image)
				spec.Containers[c] = conSpec
			}
			for s, con := range con.Sidecars {
				if conSpec, ok := spec.Containers[c].Sidecars[s]; ok {
					conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, con.Image)
					spec.Containers[c].Sidecars[s] = conSpec
				}
			}
		}
		for c, con := range imageData.Jobs {
			if conSpec, ok := spec.Jobs[c]; ok {
				conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, con.Image)
				spec.Jobs[c] = conSpec
			}
			for s, con := range con.Sidecars {
				if conSpec, ok := spec.Jobs[c].Sidecars[s]; ok {
					conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, con.Image)
					spec.Jobs[c].Sidecars[s] = conSpec
				}
			}
		}
		for i, img := range imageData.Images {
			if imgSpec, ok := spec.Images[i]; ok {
				imgSpec.Image, imgSpec.Build = assignImage(imgSpec.Image, imgSpec.Build, img.Image)
				spec.Images[i] = imgSpec
			}
		}
		for i, acorn := range imageData.Acorns {
			if acornSpec, ok := spec.Acorns[i]; ok {
				acornSpec.Image, acornSpec.Build = assignAcornImage(acornSpec.Image, acornSpec.Build, acorn.Image)
				spec.Acorns[i] = acornSpec
			}
		}
	}

	return spec, nil
}

func addContainerFiles(fileSet map[string]bool, builds map[string]v1.ContainerImageBuilderSpec, cwd string) {
	for _, build := range builds {
		addContainerFiles(fileSet, build.Sidecars, cwd)
		if build.Build == nil || build.Build.BaseImage != "" {
			continue
		}
		fileSet[filepath.Join(cwd, build.Build.Dockerfile)] = true
	}
}

func addAcorns(fileSet map[string]bool, builds map[string]v1.AcornBuilderSpec, cwd string) {
	for _, build := range builds {
		if build.Build == nil {
			continue
		}
		data, err := cue.ReadCUE(filepath.Join(cwd, build.Build.Acornfile))
		if err != nil {
			return
		}

		fileSet[filepath.Join(cwd, build.Build.Acornfile)] = true

		appDef, err := NewAppDefinition(data)
		if err != nil {
			return
		}
		files, err := appDef.WatchFiles(filepath.Join(cwd, build.Build.Context))
		if err != nil {
			return
		}
		for _, f := range files {
			fileSet[f] = true
		}
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
	addAcorns(fileSet, spec.Acorns, cwd)

	for k := range fileSet {
		result = append(result, k)
	}
	sort.Strings(result)
	return result, nil
}

func (a *AppDefinition) BuilderSpec() (*v1.BuilderSpec, error) {
	app, err := a.ctx.Value()
	if err != nil {
		return nil, err
	}

	spec := &v1.BuilderSpec{}
	return spec, a.ctx.Decode(app, spec)
}

func AppImageFromTar(reader io.Reader) (*v1.AppImage, error) {
	tar := tar.NewReader(reader)
	result := &v1.AppImage{}
	for {
		header, err := tar.Next()
		if err == io.EOF {
			break
		}

		if header.Name == AcornCueFile {
			data, err := io.ReadAll(tar)
			if err != nil {
				return nil, err
			}
			result.Acornfile = string(data)
		} else if header.Name == ImageDataFile {
			err := json.NewDecoder(tar).Decode(&result.ImageData)
			if err != nil {
				return nil, err
			}
		} else if header.Name == BuildDataFile {
			result.BuildArgs = map[string]any{}
			err := json.NewDecoder(tar).Decode(&result.BuildArgs)
			if err != nil {
				return nil, err
			}
		}
	}

	if result.Acornfile == "" {
		return nil, fmt.Errorf("invalid image no Acornfile found")
	}

	return result, nil
}
