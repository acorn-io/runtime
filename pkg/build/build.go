package build

import (
	"context"
	"io/ioutil"
	"os"
	"sort"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/build/buildkit"
	"github.com/ibuildthecloud/herd/pkg/streams"
)

type Options struct {
	Cwd     string
	Streams *streams.Output
}

func (b *Options) Complete() (*Options, error) {
	var current Options
	if b != nil {
		current = *b
	}
	if current.Cwd == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		current.Cwd = pwd
	}
	if current.Streams == nil {
		current.Streams = streams.CurrentOutput()
	}
	return &current, nil
}

func Build(ctx context.Context, file string, opts *Options) (*v1.AppImage, error) {
	opts, err := opts.Complete()
	if err != nil {
		return nil, err
	}

	fileData, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	appDefinition, err := appdefinition.NewAppDefinition(fileData)
	if err != nil {
		return nil, err
	}

	buildSpec, err := appDefinition.BuildSpec()
	if err != nil {
		return nil, err
	}

	imageData, err := FromSpec(ctx, opts.Cwd, *buildSpec, *opts.Streams)
	appImage := &v1.AppImage{
		Herdfile:  string(fileData),
		ImageData: imageData,
	}
	if err != nil {
		return nil, err
	}

	id, err := FromAppImage(ctx, appImage, *opts.Streams)
	if err != nil {
		return nil, err
	}
	appImage.ID = id

	return appImage, nil
}

func FromSpec(ctx context.Context, cwd string, spec v1.BuildSpec, streams streams.Output) (v1.ImageData, error) {
	data := v1.ImageData{
		Containers: map[string]v1.ContainerData{},
		Images:     map[string]v1.ContainerData{},
	}

	var containerKeys []string
	for k := range spec.Containers {
		containerKeys = append(containerKeys, k)
	}
	sort.Strings(containerKeys)

	for _, key := range containerKeys {
		container := spec.Containers[key]
		if container.Image != "" || container.Build == nil {
			continue
		}

		id, err := FromBuild(ctx, cwd, *container.Build, streams.Streams())
		if err != nil {
			return data, err
		}

		data.Containers[key] = v1.ContainerData{
			Image: id,
		}
	}

	var imageKeys []string
	for k := range spec.Images {
		imageKeys = append(imageKeys, k)
	}
	sort.Strings(imageKeys)

	for _, key := range imageKeys {
		image := spec.Images[key]
		if image.Image != "" || image.Build == nil {
			continue
		}

		id, err := FromBuild(ctx, cwd, *image.Build, streams.Streams())
		if err != nil {
			return data, err
		}

		data.Images[key] = v1.ContainerData{
			Image: id,
		}
	}

	return data, nil
}

func FromBuild(ctx context.Context, cwd string, build v1.Build, streams streams.Streams) (string, error) {
	if build.Dockerfile == "" {
		build.Dockerfile = "Dockerfile"
	}

	if build.Context == "" {
		build.Context = "."
	}

	return buildkit.Build(ctx, cwd, build, streams)
}
