package build

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	v1 "github.com/ibuildthecloud/herd/pkg/api/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/appimage"
	"github.com/ibuildthecloud/herd/pkg/streams"
)

type Opts struct {
	Cwd     string
	Streams *streams.Output
}

func (b *Opts) complete() (*Opts, error) {
	current := b
	if b == nil {
		current = &Opts{}
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
	return current, nil
}

func Build(ctx context.Context, file string, opts *Opts) (*appimage.AppImage, error) {
	opts, err := opts.complete()
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
	appImage := &appimage.AppImage{
		Herdfile:  fileData,
		ImageData: imageData,
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
	}

	var containerKeys []string
	for k := range spec.Containers {
		containerKeys = append(containerKeys, k)
	}
	sort.Strings(containerKeys)

	for _, key := range containerKeys {
		container := spec.Containers[key]
		if container.Image != "" {
			continue
		}

		id, err := FromBuild(ctx, cwd, container.Build, streams.Streams())
		if err != nil {
			return data, err
		}

		data.Containers[key] = v1.ContainerData{
			Image: id,
		}
	}

	return data, nil
}

func FromBuild(ctx context.Context, cwd string, build v1.Build, streams streams.Streams) (string, error) {
	iidfile, err := ioutil.TempFile("", "herd-idfile-")
	if err != nil {
		return "", err
	}
	defer os.Remove(iidfile.Name())

	if err := iidfile.Close(); err != nil {
		return "", err
	}

	dockerfile := build.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	context := build.Context
	if context == "" {
		context = "."
	}

	cmd := exec.CommandContext(ctx, "docker", "build",
		"--iidfile", iidfile.Name(),
		"-f", dockerfile, context)
	cmd.Dir = cwd
	cmd.Stdin = streams.In
	cmd.Stdout = streams.Out
	cmd.Stderr = streams.Err
	if err := cmd.Run(); err != nil {
		return "", err
	}

	id, err := ioutil.ReadFile(iidfile.Name())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(id)), nil
}
