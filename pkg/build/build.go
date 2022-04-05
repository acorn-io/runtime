package build

import (
	"context"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/ibuildthecloud/baaah/pkg/typed"
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

	buildSpec, err := appDefinition.BuilderSpec()
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

func buildContainers(ctx context.Context, cwd string, streams streams.Output, containers map[string]v1.ContainerImageBuilderSpec) (map[string]v1.ContainerData, error) {
	result := map[string]v1.ContainerData{}

	for _, entry := range typed.Sorted(containers) {
		key, container := entry.Key, entry.Value
		if container.Image != "" && container.Build == nil {
			// this is a copy, it's fine to modify it
			container.Build = &v1.Build{
				BaseImage: container.Image,
			}
		}

		id, err := FromBuild(ctx, cwd, *container.Build, streams.Streams())
		if err != nil {
			return nil, err
		}

		result[key] = v1.ContainerData{
			Image:    id,
			Sidecars: map[string]v1.ImageData{},
		}

		var sidecarKeys []string
		for k := range container.Sidecars {
			sidecarKeys = append(sidecarKeys, k)
		}
		sort.Strings(sidecarKeys)

		for _, entry := range typed.Sorted(container.Sidecars) {
			sidecarKey, sidecar := entry.Key, entry.Value
			if sidecar.Image != "" || sidecar.Build == nil {
				// this is a copy, it's fine to modify it
				sidecar.Build = &v1.Build{
					BaseImage: sidecar.Image,
				}
			}

			id, err := FromBuild(ctx, cwd, *sidecar.Build, streams.Streams())
			if err != nil {
				return nil, err
			}
			result[key].Sidecars[sidecarKey] = v1.ImageData{
				Image: id,
			}
		}
	}

	return result, nil
}

func buildImages(ctx context.Context, cwd string, streams streams.Output, images map[string]v1.ImageBuilderSpec) (map[string]v1.ImageData, error) {
	result := map[string]v1.ImageData{}

	for _, entry := range typed.Sorted(images) {
		key, image := entry.Key, entry.Value
		if image.Image != "" || image.Build == nil {
			image.Build = &v1.Build{
				BaseImage: image.Image,
			}
		}

		id, err := FromBuild(ctx, cwd, *image.Build, streams.Streams())
		if err != nil {
			return nil, err
		}

		result[key] = v1.ImageData{
			Image: id,
		}
	}

	return result, nil
}

func FromSpec(ctx context.Context, cwd string, spec v1.BuilderSpec, streams streams.Output) (v1.ImagesData, error) {
	var (
		err  error
		data = v1.ImagesData{
			Images: map[string]v1.ImageData{},
		}
	)

	data.Containers, err = buildContainers(ctx, cwd, streams, spec.Containers)
	if err != nil {
		return data, err
	}

	data.Jobs, err = buildContainers(ctx, cwd, streams, spec.Jobs)
	if err != nil {
		return data, err
	}

	data.Images, err = buildImages(ctx, cwd, streams, spec.Images)
	if err != nil {
		return data, err
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

	if build.BaseImage != "" || len(build.ContextDirs) > 0 {
		return buildWithContext(ctx, cwd, build, streams)
	}

	return builder(ctx, cwd, build, streams)
}

func builder(ctx context.Context, cwd string, build v1.Build, streams streams.Streams) (string, error) {
	return buildkit.Build(ctx, cwd, build, streams)
}

func buildWithContext(ctx context.Context, cwd string, build v1.Build, streams streams.Streams) (string, error) {
	var (
		baseImage = build.BaseImage
		err       error
	)
	if baseImage == "" {
		baseImage, err = builder(ctx, cwd, build.BaseBuild(), streams)
		if err != nil {
			return "", err
		}
	}
	dockerfile, err := ioutil.TempFile("", "herd-dockerfile-")
	if err != nil {
		return "", err
	}
	defer func() {
		dockerfile.Close()
		os.Remove(dockerfile.Name())
	}()

	_, err = dockerfile.WriteString(toContextCopyDockerFile(baseImage, build.ContextDirs))
	if err != nil {
		return "", err
	}

	if err := dockerfile.Close(); err != nil {
		return "", err
	}

	return builder(ctx, "", v1.Build{
		Context:    cwd,
		Dockerfile: dockerfile.Name(),
	}, streams)
}

func toContextCopyDockerFile(baseImage string, contextDirs map[string]string) string {
	buf := strings.Builder{}
	buf.WriteString("FROM ")
	buf.WriteString(baseImage)
	buf.WriteString("\n")
	for _, to := range typed.SortedKeys(contextDirs) {
		from := contextDirs[to]
		buf.WriteString("COPY --link \"")
		buf.WriteString(from)
		buf.WriteString("\" \"")
		buf.WriteString(to)
		buf.WriteString("\"\n")
	}
	return buf.String()
}
