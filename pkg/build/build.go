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
	"github.com/acorn-io/acorn/pkg/buildclient"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

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

func Build(ctx context.Context, messages buildclient.Messages, pushRepo string, opts *v1.AcornImageBuildInstanceSpec, keychain authn.Keychain, remoteOpts ...remote.Option) (*v1.AppImage, error) {
	appDefinition, err := appdefinition.NewAppDefinition([]byte(opts.Acornfile))
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

	imageData, err := FromSpec(ctx, pushRepo, *buildSpec, messages, keychain, remoteOpts)
	appImage := &v1.AppImage{
		Acornfile: opts.Acornfile,
		ImageData: imageData,
		BuildArgs: buildArgs,
		VCS:       opts.VCS,
	}
	if err != nil {
		return nil, err
	}

	id, err := FromAppImage(ctx, pushRepo, appImage, messages, &AppImageOptions{
		Keychain:      keychain,
		RemoteOptions: remoteOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to finalize app image: %w", err)
	}
	appImage.ID = id
	appImage.Digest = "sha256:" + id

	return appImage, nil
}

func buildContainers(ctx context.Context, pushRepo string, buildCache *buildCache, platforms []v1.Platform, messages buildclient.Messages, containers map[string]v1.ContainerImageBuilderSpec, keychain authn.Keychain, opts []remote.Option) (map[string]v1.ContainerData, error) {
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

		id, err := fromBuild(ctx, pushRepo, buildCache, platforms, *container.Build, messages, keychain, opts)
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

			id, err := fromBuild(ctx, pushRepo, buildCache, platforms, *sidecar.Build, messages, keychain, opts)
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

func buildImages(ctx context.Context, pushRepo string, buildCache *buildCache, platforms []v1.Platform, messages buildclient.Messages, images map[string]v1.ImageBuilderSpec, keychain authn.Keychain, opts []remote.Option) (map[string]v1.ImageData, error) {
	result := map[string]v1.ImageData{}

	for _, entry := range typed.Sorted(images) {
		key, image := entry.Key, entry.Value
		if image.Image != "" || image.Build == nil {
			image.Build = &v1.Build{
				BaseImage: image.Image,
			}
		}

		id, err := fromBuild(ctx, pushRepo, buildCache, platforms, *image.Build, messages, keychain, opts)
		if err != nil {
			return nil, err
		}

		result[key] = v1.ImageData{
			Image: id,
		}
	}

	return result, nil
}

func FromSpec(ctx context.Context, pushRepo string, spec v1.BuilderSpec, messages buildclient.Messages, keychain authn.Keychain, opts []remote.Option) (v1.ImagesData, error) {
	var (
		err  error
		data = v1.ImagesData{
			Images: map[string]v1.ImageData{},
		}
	)

	buildCache := &buildCache{}

	data.Containers, err = buildContainers(ctx, pushRepo, buildCache, spec.Platforms, messages, spec.Containers, keychain, opts)
	if err != nil {
		return data, err
	}

	data.Jobs, err = buildContainers(ctx, pushRepo, buildCache, spec.Platforms, messages, spec.Jobs, keychain, opts)
	if err != nil {
		return data, err
	}

	data.Images, err = buildImages(ctx, pushRepo, buildCache, spec.Platforms, messages, spec.Images, keychain, opts)
	if err != nil {
		return data, err
	}

	return data, nil
}

func fromBuild(ctx context.Context, pushRepo string, buildCache *buildCache, platforms []v1.Platform, build v1.Build, messages buildclient.Messages, keychain authn.Keychain, opts []remote.Option) (id string, err error) {
	id, err = buildCache.Get(build, platforms)
	if err != nil || id != "" {
		return id, err
	}

	defer func() {
		if err == nil && id != "" {
			buildCache.Store(build, platforms, id)
		}
	}()

	if build.Dockerfile == "" {
		build.Dockerfile = "Dockerfile"
	}

	if build.Context == "" {
		build.Context = "."
	}

	if build.BaseImage != "" || len(build.ContextDirs) > 0 {
		return buildWithContext(ctx, pushRepo, platforms, build, messages, keychain, opts)
	}

	return buildImageAndManifest(ctx, pushRepo, platforms, build, messages, keychain, opts)
}

func buildImageNoManifest(ctx context.Context, pushRepo string, cwd string, build v1.Build, messages buildclient.Messages, keychain authn.Keychain) (string, error) {
	_, ids, err := buildkit.Build(ctx, pushRepo, cwd, nil, build, messages, keychain)
	if err != nil {
		return "", err
	}
	return ids[0], nil
}

func buildImageAndManifest(ctx context.Context, pushRepo string, platforms []v1.Platform, build v1.Build, messages buildclient.Messages, keychain authn.Keychain, opts []remote.Option) (string, error) {
	platforms, ids, err := buildkit.Build(ctx, pushRepo, "", platforms, build, messages, keychain)
	if err != nil {
		return "", err
	}

	if len(ids) == 1 {
		return ids[0], nil
	}

	return createManifest(ids, platforms, opts)
}

func buildWithContext(ctx context.Context, pushRepo string, platforms []v1.Platform, build v1.Build, messages buildclient.Messages, keychain authn.Keychain, opts []remote.Option) (string, error) {
	var (
		baseImage = build.BaseImage
	)

	if baseImage == "" {
		newImage, err := buildImageAndManifest(ctx, pushRepo, platforms, build.BaseBuild(), messages, keychain, opts)
		if err != nil {
			return "", err
		}
		baseImage = newImage
	}

	return buildImageAndManifest(ctx, pushRepo, platforms, v1.Build{
		Context:            ".",
		Dockerfile:         "Dockerfile",
		DockerfileContents: toContextCopyDockerFile(baseImage, build.ContextDirs),
	}, messages, keychain, opts)
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

func (b *buildCache) toKey(platforms []v1.Platform, build v1.Build) (string, error) {
	data, err := json.Marshal(map[string]interface{}{
		"platforms": platforms,
		"build":     build,
	})
	return string(data), err
}

func (b *buildCache) Get(build v1.Build, platforms []v1.Platform) (string, error) {
	key, err := b.toKey(platforms, build)
	if err != nil {
		// ignore error and return as cache miss
		return "", nil
	}
	return b.cache[key], nil
}

func (b *buildCache) Store(build v1.Build, platforms []v1.Platform, id string) {
	key, err := b.toKey(platforms, build)
	if err != nil {
		// ignore error and return as cache miss
		return
	}
	if b.cache == nil {
		b.cache = map[string]string{}
	}
	b.cache[key] = id
}
