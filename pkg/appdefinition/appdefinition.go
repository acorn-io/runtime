package appdefinition

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/aml"
	"github.com/acorn-io/aml/pkg/cue"
	"github.com/acorn-io/baaah/pkg/typed"
	"sigs.k8s.io/yaml"
)

const (
	AcornCueFile  = "Acornfile"
	ImageDataFile = "images.json"
	VCSDataFile   = "vcs.json"
	BuildDataFile = "build.json"
)

var (
	ErrInvalidInput = errors.New("invalid input")
)

type AppDefinition struct {
	data         []byte
	imageDatas   []v1.ImagesData
	hasImageData bool
	args         map[string]any
	profiles     []string
}

func FromAppImage(appImage *v1.AppImage) (*AppDefinition, error) {
	appDef, err := NewAppDefinition([]byte(appImage.Acornfile))
	if err != nil {
		return nil, err
	}

	appDef = appDef.WithImageData(appImage.ImageData)
	return appDef, err
}

func (a *AppDefinition) clone() AppDefinition {
	return AppDefinition{
		data:         a.data,
		imageDatas:   a.imageDatas,
		hasImageData: a.hasImageData,
		args:         a.args,
		profiles:     a.profiles,
	}
}

func (a *AppDefinition) WithImageData(imageData v1.ImagesData) *AppDefinition {
	result := a.clone()
	result.hasImageData = true
	result.imageDatas = append(result.imageDatas, imageData)
	return &result
}

func NewAppDefinition(data []byte) (*AppDefinition, error) {
	appDef := &AppDefinition{
		data: data,
	}
	_, err := appDef.AppSpec()
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
		build = &v1.Build{
			Context:    ".",
			Dockerfile: "Dockerfile",
		}
	}
	if build.BaseImage == "" {
		build.BaseImage = originalImage
	} else if build.BaseImage == originalImage {
		build.BaseImage = image
	}
	return image, build
}

func (a *AppDefinition) WithArgs(args map[string]any, profiles []string) (*AppDefinition, map[string]any, error) {
	result := a.clone()
	result.args = args
	result.profiles = profiles

	args, err := result.newDecoder().ComputedArgs()
	return &result, args, err
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
	appSpec, err := a.AppSpec()
	if err != nil {
		return "", err
	}
	app, err := json.MarshalIndent(appSpec, "", "  ")
	return string(app), err
}

func (a *AppDefinition) newDecoder() *aml.Decoder {
	return aml.NewDecoder(bytes.NewReader(a.data), aml.Options{
		Args:      a.args,
		Profiles:  a.profiles,
		Acornfile: true,
	})
}

func (a *AppDefinition) imagesData() (result v1.ImagesData) {
	for _, imageData := range a.imageDatas {
		result.Containers = typed.Concat(result.Containers, imageData.Containers)
		result.Jobs = typed.Concat(result.Jobs, imageData.Jobs)
		result.Images = typed.Concat(result.Images, imageData.Images)
		result.Acorns = typed.Concat(result.Acorns, imageData.Acorns)
		result.Builds = append(result.Builds, imageData.Builds...)
	}
	return
}

func (a *AppDefinition) AppSpec() (*v1.AppSpec, error) {
	spec := &v1.AppSpec{}
	if err := a.newDecoder().Decode(spec); err != nil {
		return nil, err
	}

	if !a.hasImageData {
		return spec, nil
	}

	imagesData := a.imagesData()

	for containerName, conSpec := range spec.Containers {
		if image, ok := GetImageReferenceForServiceName(containerName, spec, imagesData); ok {
			conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, image)
		} else {
			return nil, fmt.Errorf("failed to find image for container [%s] in Acornfile", containerName)
		}
		for sidecarName, sidecarSpec := range conSpec.Sidecars {
			if image, ok := GetImageReferenceForServiceName(containerName+"."+sidecarName, spec, imagesData); ok {
				sidecarSpec.Image, sidecarSpec.Build = assignImage(sidecarSpec.Image, sidecarSpec.Build, image)
				conSpec.Sidecars[sidecarName] = sidecarSpec
			} else {
				return nil, fmt.Errorf("failed to find image for sidecar [%s] in container [%s] in Acornfile", sidecarName, containerName)
			}
		}
		spec.Containers[containerName] = conSpec
	}

	for containerName, conSpec := range spec.Jobs {
		if image, ok := GetImageReferenceForServiceName(containerName, spec, imagesData); ok {
			conSpec.Image, conSpec.Build = assignImage(conSpec.Image, conSpec.Build, image)
		} else {
			return nil, fmt.Errorf("failed to find image for job [%s] in Acornfile", containerName)
		}
		for sidecarName, sidecarSpec := range conSpec.Sidecars {
			if image, ok := GetImageReferenceForServiceName(containerName+"."+sidecarName, spec, imagesData); ok {
				sidecarSpec.Image, sidecarSpec.Build = assignImage(sidecarSpec.Image, sidecarSpec.Build, image)
				conSpec.Sidecars[sidecarName] = sidecarSpec
			} else {
				return nil, fmt.Errorf("failed to find image for sidecar [%s] in job [%s] in Acornfile", sidecarName, containerName)
			}
		}
		spec.Jobs[containerName] = conSpec
	}

	for imageName, imgSpec := range spec.Images {
		if image, ok := GetImageReferenceForServiceName(imageName, spec, imagesData); ok {
			if imgSpec.AcornBuild != nil {
				imgSpec.Image, imgSpec.AcornBuild = assignAcornImage(imgSpec.Image, imgSpec.AcornBuild, image)
			} else {
				imgSpec.Image, imgSpec.Build = assignImage(imgSpec.Image, imgSpec.Build, image)
			}
		} else {
			return nil, fmt.Errorf("failed to find image for image definition [%s] in Acornfile", imageName)
		}
		spec.Images[imageName] = imgSpec
	}

	for acornName, acornSpec := range spec.Acorns {
		if image, ok := GetImageReferenceForServiceName(acornName, spec, imagesData); ok {
			acornSpec.Image, acornSpec.Build = assignAcornImage(acornSpec.Image, acornSpec.Build, image)
		} else {
			return nil, fmt.Errorf("failed to find image for acorn [%s] in Acornfile", acornName)
		}
		spec.Acorns[acornName] = acornSpec
	}

	for serviceName, serviceSpec := range spec.Services {
		if serviceSpec.Image == "" && serviceSpec.Build == nil {
			continue
		}
		if image, ok := GetImageReferenceForServiceName(serviceName, spec, imagesData); ok {
			serviceSpec.Image, serviceSpec.Build = assignAcornImage(serviceSpec.Image, serviceSpec.Build, image)
		} else {
			return nil, fmt.Errorf("failed to find image for service [%s] in Acornfile", serviceName)
		}
		spec.Services[serviceName] = serviceSpec
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
		fileSet[filepath.Join(filepath.Dir(filepath.Join(cwd, build.Build.Dockerfile)), ".dockerignore")] = true
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
		if build.ContainerBuild == nil {
			if build.AcornBuild != nil {
				fileSet[filepath.Join(cwd, build.AcornBuild.Acornfile)] = true
			}
		} else {
			fileSet[filepath.Join(cwd, build.ContainerBuild.Dockerfile)] = true
		}
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
	addAcorns(fileSet, spec.Services, cwd)
	addAcorns(fileSet, spec.Acorns, cwd)

	for k := range fileSet {
		result = append(result, k)
	}
	sort.Strings(result)
	return result, nil
}

func (a *AppDefinition) BuilderSpec() (*v1.BuilderSpec, error) {
	spec := &v1.BuilderSpec{}
	return spec, a.newDecoder().Decode(spec)
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
