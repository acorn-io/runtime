package appdefinition

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	cue2 "cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	cue_mod "github.com/acorn-io/acorn/cue.mod"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/acorn-io/acorn/schema"
	"github.com/mitchellh/hashstructure"
	"sigs.k8s.io/yaml"
)

const (
	AcornCueFile  = "Acornfile"
	ImageDataFile = "images.json"
	VCSDataFile   = "vcs.json"
	BuildDataFile = "build.json"
	Schema        = "github.com/acorn-io/acorn/schema/v1"
	AppType       = "#App"
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

var ErrBuildHashImgNotFound error = errors.New("no image found for given build hash")

func FindImage(build *v1.Build, imageDatas []v1.ImagesData, originalImage string) (string, error) {

	if build == nil {
		build = &v1.Build{}
	}
	if build.BaseImage == "" {
		build.BaseImage = originalImage
	}

	buildhash, err := hashstructure.Hash(build, nil)
	if err != nil {
		return "", err
	}

	hash := strconv.FormatUint(buildhash, 10)

	for _, imageData := range imageDatas {
		if img, ok := imageData[hash]; ok {
			return img, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrBuildHashImgNotFound, hash)

}

func (a *AppDefinition) AppSpec() (*v1.AppSpec, error) {
	app, err := a.ctx.Value()
	if err != nil {
		return nil, err
	}

	objs := map[string]any{}
	for _, key := range []string{"containers", "jobs", "acorns", "secrets", "volumes", "images", "routers", "labels", "annotations"} {
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

	// Containers + Container Sidecars
	for conName, conSpec := range spec.Containers {
		img, err := FindImage(conSpec.Build, a.imageDatas, conSpec.Image)
		if err != nil && !errors.Is(err, ErrBuildHashImgNotFound) {
			return nil, err
		}

		conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, img)
		spec.Containers[conName] = conSpec

		for sName, sSpec := range conSpec.Sidecars {
			img, err := FindImage(sSpec.Build, a.imageDatas, sSpec.Image)
			if err != nil && !errors.Is(err, ErrBuildHashImgNotFound) {
				return nil, err
			}
			conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, img)
			spec.Containers[conName].Sidecars[sName] = conSpec
		}
	}

	// Jobs + Job Sidecars
	for conName, conSpec := range spec.Jobs {
		img, err := FindImage(conSpec.Build, a.imageDatas, conSpec.Image)
		if err != nil && !errors.Is(err, ErrBuildHashImgNotFound) {
			return nil, err
		}

		conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, img)
		spec.Jobs[conName] = conSpec

		for sName, sSpec := range conSpec.Sidecars {
			img, err := FindImage(sSpec.Build, a.imageDatas, sSpec.Image)
			if err != nil && !errors.Is(err, ErrBuildHashImgNotFound) {
				return nil, err
			}
			conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, img)
			spec.Containers[conName].Sidecars[sName] = conSpec
		}
	}

	// Images
	for conName, conSpec := range spec.Images {
		img, err := FindImage(conSpec.Build, a.imageDatas, conSpec.Image)
		if err != nil && !errors.Is(err, ErrBuildHashImgNotFound) {
			return nil, err
		}

		conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, img)
		spec.Images[conName] = conSpec
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
	addContainerFiles(fileSet, spec.Jobs, cwd)
	addFiles(fileSet, spec.Images, cwd)

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
		} else if header.Name == VCSDataFile {
			err := json.NewDecoder(tar).Decode(&result.VCS)
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
