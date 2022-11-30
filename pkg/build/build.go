package build

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/typed"
	imagename "github.com/google/go-containerregistry/pkg/name"
)

type Options struct {
	Client    client.Client
	Cwd       string
	Platforms []v1.Platform
	Args      map[string]any
	Profiles  []string
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
	return filepath.Join(cwd, "Acornfile")
}

func ResolveFile(file, cwd string) string {
	if file == "DIRECTORY/Acornfile" {
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

	vcs := vcs(filepath.Dir(file))

	appDefinition, err := appdefinition.NewAppDefinition(fileData)
	if err != nil {
		return nil, err
	}

	appDefinition, buildArgs, err := appDefinition.WithArgs(opts.Args, append([]string{"build?"}, opts.Profiles...))
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
		Acornfile: string(fileData),
		ImageData: imageData,
		BuildArgs: buildArgs,
		VCS:       vcs,
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

func buildContainers(ctx context.Context, c client.Client, buildCache *buildCache, cwd string, platforms []v1.Platform, streams streams.Output, containers map[string]v1.ContainerImageBuilderSpec) (map[string]v1.ContainerData, error) {
	result := map[string]v1.ContainerData{}

	for _, entry := range typed.Sorted(containers) {
		key, container := entry.Key, entry.Value

		if container.Image == "" && container.Build == nil {
			return nil, fmt.Errorf("either image or build field must be set")
		}

		if container.Image != "" && container.Build == nil {
			// this is a copy, it's fine to modify it
			container.Build = &v1.Build{
				BaseImage: container.Image,
			}
		}

		id, err := fromBuild(ctx, c, buildCache, cwd, platforms, *container.Build, streams.Streams())
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
				if sidecar.Build == nil {
					sidecar.Build = &v1.Build{
						BaseImage: sidecar.Image,
					}
				} else {
					sidecar.Build.BaseImage = sidecar.Image
				}
			}

			id, err := fromBuild(ctx, c, buildCache, cwd, platforms, *sidecar.Build, streams.Streams())
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

func buildImages(ctx context.Context, c client.Client, buildCache *buildCache, cwd string, platforms []v1.Platform, streams streams.Output, images map[string]v1.ImageBuilderSpec) (map[string]v1.ImageData, error) {
	result := map[string]v1.ImageData{}

	for _, entry := range typed.Sorted(images) {
		key, image := entry.Key, entry.Value
		if image.Image != "" || image.Build == nil {
			image.Build = &v1.Build{
				BaseImage: image.Image,
			}
		}

		id, err := fromBuild(ctx, c, buildCache, cwd, platforms, *image.Build, streams.Streams())
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

	buildCache := &buildCache{}

	data.Containers, err = buildContainers(ctx, c, buildCache, cwd, spec.Platforms, streams, spec.Containers)
	if err != nil {
		return data, err
	}

	data.Jobs, err = buildContainers(ctx, c, buildCache, cwd, spec.Platforms, streams, spec.Jobs)
	if err != nil {
		return data, err
	}

	data.Images, err = buildImages(ctx, c, buildCache, cwd, spec.Platforms, streams, spec.Images)
	if err != nil {
		return data, err
	}

	return data, nil
}

func fromBuild(ctx context.Context, c client.Client, buildCache *buildCache, cwd string, platforms []v1.Platform, build v1.Build, streams streams.Streams) (id string, err error) {
	id, err = buildCache.Get(cwd, build, platforms)
	if err != nil || id != "" {
		return id, err
	}

	defer func() {
		if err == nil && id != "" {
			buildCache.Store(cwd, build, platforms, id)
		}
	}()

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

	if len(ids) == 1 {
		return ids[0], nil
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
	dockerfile, err := os.CreateTemp("", "acorn-dockerfile-")
	if err != nil {
		return "", err
	}
	defer func() {
		dockerfile.Close()
		os.Remove(dockerfile.Name())
	}()

	for _, dir := range build.ContextDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// don't blindly mkdirall because this could actually be a file
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				return "", fmt.Errorf("creating dir %s: %w", dir, err)
			}
		}
	}

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

type buildCache struct {
	cache map[string]string
}

func (b *buildCache) toKey(cwd string, platforms []v1.Platform, build v1.Build) (string, error) {
	data, err := json.Marshal(map[string]interface{}{
		"cwd":       cwd,
		"platforms": platforms,
		"build":     build,
	})
	return string(data), err
}

func (b *buildCache) Get(cwd string, build v1.Build, platforms []v1.Platform) (string, error) {
	key, err := b.toKey(cwd, platforms, build)
	if err != nil {
		// ignore error and return as cache miss
		return "", nil
	}
	return b.cache[key], nil
}

func (b *buildCache) Store(cwd string, build v1.Build, platforms []v1.Platform, id string) {
	key, err := b.toKey(cwd, platforms, build)
	if err != nil {
		// ignore error and return as cache miss
		return
	}
	if b.cache == nil {
		b.cache = map[string]string{}
	}
	b.cache[key] = id
}
