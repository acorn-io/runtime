package build

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/acorn-io/acorn/pkg/remoteopts"
	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/typed"
	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Options struct {
	Client    client.Client
	Cwd       string
	Platforms []v1.Platform
	Args      map[string]interface{}
	Streams   *streams.Output
	FullTag   bool
}

func (b *Options) Complete() (*Options, error) {
	var (
		current Options
		err     error
	)
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
	if current.Client == nil {
		current.Client, err = client.Default()
		if err != nil {
			return nil, err
		}
	}
	return &current, nil
}

func FindAcornCue(cwd string) string {
	for _, ext := range []string{"cue", "yaml", "json"} {
		f := filepath.Join(cwd, "acorn."+ext)
		if _, err := os.Stat(f); err == nil {
			return f
		}
	}
	return filepath.Join(cwd, "acorn.cue")
}

func ResolveFile(file, cwd string) string {
	if file == "DIRECTORY/acorn.cue" {
		return FindAcornCue(cwd)
	}
	return file
}

func ResolveAndParse(file, cwd string) (*appdefinition.AppDefinition, error) {
	file = ResolveFile(file, cwd)

	fileData, err := cue.ReadCUE(file)
	if err != nil {
		return nil, err
	}

	return appdefinition.NewAppDefinition(fileData)
}

func Build(ctx context.Context, file string, opts *Options) (*v1.AppImage, error) {
	opts, err := opts.Complete()
	if err != nil {
		return nil, err
	}

	file = ResolveFile(file, opts.Cwd)

	fileData, err := cue.ReadCUE(file)
	if err != nil {
		return nil, err
	}

	appDefinition, err := appdefinition.NewAppDefinition(fileData)
	if err != nil {
		return nil, err
	}

	appDefinition, err = appDefinition.WithBuildArgs(opts.Args)
	if err != nil {
		return nil, err
	}

	buildSpec, err := appDefinition.BuilderSpec()
	if err != nil {
		return nil, err
	}
	buildSpec.Platforms = opts.Platforms

	imageData, err := FromSpec(ctx, opts.Client, opts.Cwd, *buildSpec, *opts.Streams)
	appImage := &v1.AppImage{
		Acornfile:   string(fileData),
		ImageData:   imageData,
		BuildParams: opts.Args,
	}
	if err != nil {
		return nil, err
	}

	id, err := FromAppImage(ctx, opts.Client, appImage, *opts.Streams, &AppImageOptions{
		FullTag: opts.FullTag,
	})
	if err != nil {
		return nil, err
	}
	appImage.ID = id

	return appImage, nil
}

func buildContainers(ctx context.Context, c client.Client, cwd string, platforms []v1.Platform, streams streams.Output, containers map[string]v1.ContainerImageBuilderSpec) (map[string]v1.ContainerData, error) {
	result := map[string]v1.ContainerData{}

	for _, entry := range typed.Sorted(containers) {
		key, container := entry.Key, entry.Value
		if container.Image != "" && container.Build == nil {
			// this is a copy, it's fine to modify it
			container.Build = &v1.Build{
				BaseImage: container.Image,
			}
		}

		id, err := FromBuild(ctx, c, cwd, platforms, *container.Build, streams.Streams())
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

			id, err := FromBuild(ctx, c, cwd, platforms, *sidecar.Build, streams.Streams())
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

func buildAcorns(ctx context.Context, c client.Client, cwd string, platforms []v1.Platform, streams streams.Output, acorns map[string]v1.AcornBuilderSpec) (map[string]v1.ImageData, error) {
	result := map[string]v1.ImageData{}

	for _, entry := range typed.Sorted(acorns) {
		key, acornImage := entry.Key, entry.Value
		if acornImage.Image != "" {
			tag, err := imagename.ParseReference(acornImage.Image)
			if err != nil {
				return nil, err
			}

			opts, err := remoteopts.WithClientDialer(ctx, c)
			if err != nil {
				return nil, err
			}

			index, err := remote.Index(tag, opts...)
			if err != nil {
				return nil, err
			}

			digest, err := index.Digest()
			if err != nil {
				return nil, err
			}

			result[key] = v1.ImageData{
				Image: tag.Context().Digest(digest.String()).Name(),
			}
		} else if acornImage.Build != nil {
			appImage, err := Build(ctx, filepath.Join(cwd, acornImage.Build.Acornfile), &Options{
				Client:    c,
				Cwd:       filepath.Join(cwd, acornImage.Build.Context),
				Platforms: platforms,
				Args:      acornImage.Build.BuildArgs,
				Streams:   &streams,
				FullTag:   true,
			})
			if err != nil {
				return nil, err
			}
			result[key] = v1.ImageData{
				Image: appImage.ID,
			}
		}
	}

	return result, nil
}
func buildImages(ctx context.Context, c client.Client, cwd string, platforms []v1.Platform, streams streams.Output, images map[string]v1.ImageBuilderSpec) (map[string]v1.ImageData, error) {
	result := map[string]v1.ImageData{}

	for _, entry := range typed.Sorted(images) {
		key, image := entry.Key, entry.Value
		if image.Image != "" || image.Build == nil {
			image.Build = &v1.Build{
				BaseImage: image.Image,
			}
		}

		id, err := FromBuild(ctx, c, cwd, platforms, *image.Build, streams.Streams())
		if err != nil {
			return nil, err
		}

		result[key] = v1.ImageData{
			Image: id,
		}
	}

	return result, nil
}

func FromSpec(ctx context.Context, c client.Client, cwd string, spec v1.BuilderSpec, streams streams.Output) (v1.ImagesData, error) {
	var (
		err  error
		data = v1.ImagesData{
			Images: map[string]v1.ImageData{},
		}
	)

	data.Containers, err = buildContainers(ctx, c, cwd, spec.Platforms, streams, spec.Containers)
	if err != nil {
		return data, err
	}

	data.Jobs, err = buildContainers(ctx, c, cwd, spec.Platforms, streams, spec.Jobs)
	if err != nil {
		return data, err
	}

	data.Images, err = buildImages(ctx, c, cwd, spec.Platforms, streams, spec.Images)
	if err != nil {
		return data, err
	}

	data.Acorns, err = buildAcorns(ctx, c, cwd, spec.Platforms, streams, spec.Acorns)
	if err != nil {
		return data, err
	}

	return data, nil
}

func FromBuild(ctx context.Context, c client.Client, cwd string, platforms []v1.Platform, build v1.Build, streams streams.Streams) (string, error) {
	if build.Dockerfile == "" {
		build.Dockerfile = "Dockerfile"
	}

	if build.Context == "" {
		build.Context = "."
	}

	if build.BaseImage != "" || len(build.ContextDirs) > 0 {
		return buildWithContext(ctx, c, cwd, platforms, build, streams)
	}

	return buildImageAndManifest(ctx, c, cwd, platforms, build, streams)
}

func buildImageNoManifest(ctx context.Context, c client.Client, cwd string, build v1.Build, streams streams.Streams) (string, error) {
	_, ids, err := buildkit.Build(ctx, c, cwd, nil, build, streams)
	if err != nil {
		return "", err
	}
	return ids[0], nil
}

func buildImageAndManifest(ctx context.Context, c client.Client, cwd string, platforms []v1.Platform, build v1.Build, streams streams.Streams) (string, error) {
	platforms, ids, err := buildkit.Build(ctx, c, cwd, platforms, build, streams)
	if err != nil {
		return "", err
	}

	return createManifest(ctx, c, ids, platforms)
}

func buildWithContext(ctx context.Context, c client.Client, cwd string, platforms []v1.Platform, build v1.Build, streams streams.Streams) (string, error) {
	var (
		baseImage = build.BaseImage
		err       error
	)

	if baseImage == "" {
		newImage, err := buildImageAndManifest(ctx, c, cwd, platforms, build.BaseBuild(), streams)
		if err != nil {
			return "", err
		}
		digest, err := imagename.NewDigest(newImage)
		if err != nil {
			return "", err
		}
		baseImage = strings.Replace(newImage, digest.RegistryStr(), fmt.Sprintf("127.0.0.1:%d", system.RegistryPort), 1)
	}
	dockerfile, err := ioutil.TempFile("", "acorn-dockerfile-")
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

	return buildImageAndManifest(ctx, c, "", platforms, v1.Build{
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
