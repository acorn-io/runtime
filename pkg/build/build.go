package build

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/buildclient"
	images2 "github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/aml/pkg/cue"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/authn"
	imagename "github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/uuid"
	client2 "github.com/moby/buildkit/client"
	"github.com/opencontainers/go-digest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ResolveAndParse(file string) (*appdefinition.AppDefinition, error) {
	fileData, err := cue.ReadCUE(file)
	if err != nil {
		return nil, err
	}

	return appdefinition.NewAppDefinition(fileData)
}

type buildContext struct {
	ctx           context.Context
	cwd           string
	acornfilePath string
	pushRepo      string
	opts          v1.AcornImageBuildInstanceSpec
	keychain      authn.Keychain
	remoteOpts    []remote.Option
	messages      buildclient.Messages
}

func Build(ctx context.Context, messages buildclient.Messages, pushRepo string, opts v1.AcornImageBuildInstanceSpec, keychain authn.Keychain, remoteOpts ...remote.Option) (*v1.AppImage, error) {
	remoteKc := NewRemoteKeyChain(messages, keychain)
	buildContext := &buildContext{
		ctx:        ctx,
		cwd:        "",
		pushRepo:   pushRepo,
		opts:       opts,
		keychain:   remoteKc,
		remoteOpts: append(remoteOpts, remote.WithAuthFromKeychain(remoteKc), remote.WithContext(ctx)),
		messages:   messages,
	}

	return build(buildContext)
}

func build(ctx *buildContext) (*v1.AppImage, error) {
	var (
		acornfileData []byte
		err           error
	)

	if ctx.acornfilePath == "" {
		acornfileData = []byte(ctx.opts.Acornfile)
	} else {
		acornfileData, err = getAcornfile(ctx, ctx.acornfilePath)
		if err != nil {
			return nil, err
		}
	}

	appDefinition, err := appdefinition.NewAppDefinition(acornfileData)
	if err != nil {
		return nil, err
	}

	appDefinition, buildArgs, err := appDefinition.WithArgs(ctx.opts.Args, append([]string{"build?"}, ctx.opts.Profiles...))
	if err != nil {
		return nil, err
	}

	buildSpec, err := appDefinition.BuilderSpec()
	if err != nil {
		return nil, err
	}

	imageData, err := fromSpec(ctx, *buildSpec)
	appImage := &v1.AppImage{
		Acornfile: string(acornfileData),
		ImageData: imageData,
		BuildArgs: buildArgs,
		VCS:       ctx.opts.VCS,
	}
	if err != nil {
		return nil, err
	}

	id, err := fromAppImage(ctx, appImage)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize app image: %w", err)
	}
	appImage.ID = id
	appImage.Digest = "sha256:" + id

	return appImage, nil
}

func buildContainers(ctx *buildContext, buildCache *buildCache, containers map[string]v1.ContainerImageBuilderSpec) (map[string]v1.ContainerData, []v1.BuildRecord, error) {
	var builds []v1.BuildRecord
	result := map[string]v1.ContainerData{}

	for _, entry := range typed.Sorted(containers) {
		key, container := entry.Key, entry.Value

		if container.Image == "" && container.Build == nil {
			return nil, nil, fmt.Errorf("either image or build field must be set")
		}

		if container.Image != "" && container.Build == nil {
			// this is a copy, it's fine to modify it
			container.Build = &v1.Build{
				BaseImage: container.Image,
			}
		}

		id, err := fromBuild(ctx, buildCache, *container.Build)
		if err != nil {
			return nil, nil, err
		}

		result[key] = v1.ContainerData{
			Image:    id,
			Sidecars: map[string]v1.ImageData{},
		}

		builds = append(builds, v1.BuildRecord{
			ContainerBuild: container.Normalize(),
			ImageKey:       key,
		})

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

			id, err := fromBuild(ctx, buildCache, *sidecar.Build)
			if err != nil {
				return nil, nil, err
			}
			result[key].Sidecars[sidecarKey] = v1.ImageData{
				Image: id,
			}
			builds = append(builds, v1.BuildRecord{
				ContainerBuild: sidecar.Normalize(),
				ImageKey:       key + "." + sidecarKey,
			})
		}
	}

	return result, builds, nil
}

func buildAcorns(ctx *buildContext, acorns map[string]v1.AcornBuilderSpec) (map[string]v1.ImageData, []v1.BuildRecord, error) {
	var builds []v1.BuildRecord
	result := map[string]v1.ImageData{}

	for _, entry := range typed.Sorted(acorns) {
		key, acornImage := entry.Key, entry.Value
		if acornImage.Image != "" {
			if _, auto := autoupgrade.AutoUpgradePattern(acornImage.Image); auto || acornImage.AutoUpgrade {
				// This is the one situation where ImageKey is not set
				builds = append(builds, v1.BuildRecord{
					AcornBuild: &acornImage,
				})
				continue
			}
			isRemote, err := isImageRemote(ctx, acornImage.Image)
			if err != nil {
				return nil, nil, err
			}

			var id string
			if isRemote {
				id, err = pullImage(ctx, acornImage.Image)
			} else {
				id, err = resolveLocalImage(ctx, acornImage.Image)
			}
			if err != nil {
				return nil, nil, err
			}

			result[key] = v1.ImageData{
				Image: id,
			}
			builds = append(builds, v1.BuildRecord{
				AcornBuild: acornImage.Normalize(),
				ImageKey:   key,
			})
		} else if acornImage.Build != nil {
			newCtx := *ctx
			newCtx.opts.Profiles = nil
			newCtx.opts.Args = acornImage.Build.BuildArgs
			newCtx.opts.Acornfile = ""
			newCtx.acornfilePath = filepath.Join(ctx.cwd, acornImage.Build.Acornfile)
			newCtx.cwd = filepath.Join(ctx.cwd, acornImage.Build.Context)
			appImage, err := build(&newCtx)
			if err != nil {
				return nil, nil, err
			}
			repo, err := imagename.NewRepository(ctx.pushRepo)
			if err != nil {
				return nil, nil, err
			}
			result[key] = v1.ImageData{
				Image: repo.Digest(appImage.Digest).String(),
			}
			builds = append(builds, v1.BuildRecord{
				AcornBuild:    acornImage.Normalize(),
				AcornAppImage: appImage,
				ImageKey:      key,
			})
		}
	}

	return result, builds, nil
}

func buildImages(ctx *buildContext, buildCache *buildCache, images map[string]v1.ImageBuilderSpec) (map[string]v1.ImageData, []v1.BuildRecord, error) {
	var builds []v1.BuildRecord
	result := map[string]v1.ImageData{}
	acornBuilds := map[string]v1.AcornBuilderSpec{}

	for _, entry := range typed.Sorted(images) {
		key, image := entry.Key, entry.Value
		if image.ContainerBuild == nil {
			acornBuilds[key] = v1.AcornBuilderSpec{
				Image: image.Image,
				Build: image.AcornBuild,
			}
		} else {
			if image.Image != "" {
				image.ContainerBuild = &v1.Build{
					BaseImage: image.Image,
				}
			}

			id, err := fromBuild(ctx, buildCache, *image.ContainerBuild)
			if err != nil {
				return nil, nil, err
			}

			result[key] = v1.ImageData{
				Image: id,
			}
			builds = append(builds, v1.BuildRecord{
				ImageBuild: image.Normalize(),
				ImageKey:   key,
			})
		}
	}

	acornImages, acornBuildRecords, err := buildAcorns(ctx, acornBuilds)
	if err != nil {
		return nil, nil, err
	}

	return typed.Concat(result, acornImages), append(builds, acornBuildRecords...), nil
}

func fromSpec(ctx *buildContext, spec v1.BuilderSpec) (v1.ImagesData, error) {
	var (
		err  error
		data = v1.ImagesData{
			Images: map[string]v1.ImageData{},
		}
		builds []v1.BuildRecord
	)

	buildCache := &buildCache{}

	data.Containers, builds, err = buildContainers(ctx, buildCache, spec.Containers)
	if err != nil {
		return data, err
	}
	data.Builds = append(data.Builds, builds...)

	data.Jobs, builds, err = buildContainers(ctx, buildCache, spec.Jobs)
	if err != nil {
		return data, err
	}
	data.Builds = append(data.Builds, builds...)

	data.Images, builds, err = buildImages(ctx, buildCache, spec.Images)
	if err != nil {
		return data, err
	}
	data.Builds = append(data.Builds, builds...)

	data.Acorns, builds, err = buildAcorns(ctx, typed.Concat(spec.Acorns, spec.Services))
	if err != nil {
		return data, err
	}
	data.Builds = append(data.Builds, builds...)

	return data, nil
}

func pullImage(ctx *buildContext, image string) (id string, err error) {
	ref, err := images2.ParseReferenceNoDefault(image)
	if err != nil {
		return "", err
	}

	index, err := remote.Index(ref, ctx.remoteOpts...)
	if err != nil {
		return "", err
	}

	digest, err := index.Digest()
	if err != nil {
		return "", err
	}

	pushTarget, err := imagename.ParseReference(ctx.pushRepo)
	if err != nil {
		return "", err
	}

	progress := make(chan ggcrv1.Update)
	defer progressClose(progress)
	go func() {
		printImageProgress(ctx, digest.String(), image, progress)
	}()

	pushRef := pushTarget.Context().Tag(digest.Hex)
	if err := remote.WriteIndex(pushRef, index, append(ctx.remoteOpts, remote.WithProgress(progress))...); err != nil {
		return "", err
	}

	return pushTarget.Context().Digest(digest.String()).String(), nil
}

func isImageRemote(ctx *buildContext, image string) (bool, error) {
	ref, err := images2.ParseReferenceNoDefault(image)
	if err != nil {
		return false, nil
	}

	_, err = remote.Index(ref, ctx.remoteOpts...)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func resolveLocalImage(ctx *buildContext, image string) (string, error) {
	// We need to be careful to make sure we only list ImageInstances in the namespace to which the user has access.
	// We can determine the namespace based on the pushRepo, which is in the format <registry>/acorn/<namespace>.
	// To be very safe, verify the structure of pushRepo before getting the namespace from it.
	if len(strings.Split(ctx.pushRepo, "/")) != 3 || !strings.HasPrefix(ctx.pushRepo, "127.0.0.1:") || strings.Split(ctx.pushRepo, "/")[1] != "acorn" {
		return "", fmt.Errorf("invalid pushRepo: %s", ctx.pushRepo)
	}

	namespace := strings.Split(ctx.pushRepo, "/")[2]

	c, err := k8sclient.Default()
	if err != nil {
		return "", err
	}

	imageList := v1.ImageInstanceList{}
	err = c.List(ctx.ctx, &imageList, &kclient.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return "", err
	}

	var digest string
imageLoop:
	for _, imageInstance := range imageList.Items {
		// check whether the image specified by the user matches the name (ID) of the image
		// require at least the first three characters to match
		if len(image) > 2 && strings.HasPrefix(imageInstance.Name, image) {
			digest = imageInstance.Digest
			break
		}

		// check to see if it matches any of the tags on the image
		for _, tag := range imageInstance.Tags {
			if tag == image {
				digest = imageInstance.Digest
				break imageLoop
			}
		}
	}

	if digest != "" {
		pushTarget, err := imagename.ParseReference(ctx.pushRepo)
		if err != nil {
			return "", err
		}

		return pushTarget.Context().Digest(digest).String(), nil
	}
	return "", fmt.Errorf("could not find local image %s", image)
}

func progressClose(progress chan ggcrv1.Update) {
	select {
	case <-progress:
	default:
		close(progress)
	}
}

func printImageProgress(ctx *buildContext, id, name string, progress chan ggcrv1.Update) {
	var (
		vertex    *client2.Vertex
		sessionid = uuid.New().String()
		total     int64
	)

	defer func() {
		if vertex != nil && vertex.Completed == nil {
			now := time.Now()
			vertex.Completed = &now

			_ = ctx.messages.Send(&buildclient.Message{
				StatusSessionID: sessionid,
				Status: &client2.SolveStatus{
					Statuses: []*client2.VertexStatus{
						{
							ID:        "pulling image",
							Vertex:    vertex.Digest,
							Name:      vertex.Name,
							Total:     total,
							Current:   total,
							Timestamp: time.Now(),
							Started:   vertex.Started,
							Completed: vertex.Completed,
						},
					},
					Vertexes: []*client2.Vertex{
						vertex,
					},
				},
			})

			_ = ctx.messages.Send(&buildclient.Message{
				StatusSessionID: sessionid,
				Status: &client2.SolveStatus{
					Vertexes: []*client2.Vertex{
						vertex,
					},
				},
			})
		}
	}()

	for update := range progress {
		if vertex == nil {
			now := time.Now()
			vertex = &client2.Vertex{
				Digest:  digest.Digest(id),
				Name:    name,
				Started: &now,
			}
		}

		if update.Error != nil {
			now := time.Now()
			vertex.Error = update.Error.Error()
			vertex.Completed = &now
			_ = ctx.messages.Send(&buildclient.Message{
				StatusSessionID: sessionid,
				Status: &client2.SolveStatus{
					Vertexes: []*client2.Vertex{
						vertex,
					},
				},
			})
		} else if update.Total > 0 {
			total = update.Total
			_ = ctx.messages.Send(&buildclient.Message{
				StatusSessionID: sessionid,
				Status: &client2.SolveStatus{
					Statuses: []*client2.VertexStatus{
						{
							ID:        "pulling image",
							Vertex:    vertex.Digest,
							Name:      vertex.Name,
							Total:     update.Total,
							Current:   update.Complete,
							Timestamp: time.Now(),
							Started:   vertex.Started,
							Completed: vertex.Completed,
						},
					},
					Vertexes: []*client2.Vertex{
						vertex,
					},
				},
			})
		}
	}
}

func fromBuild(ctx *buildContext, buildCache *buildCache, build v1.Build) (id string, err error) {
	id, err = buildCache.Get(build, ctx.opts.Platforms)
	if err != nil || id != "" {
		return id, err
	}

	defer func() {
		if err == nil && id != "" {
			buildCache.Store(build, ctx.opts.Platforms, id)
		}
	}()

	if build.Dockerfile == "" {
		build.Dockerfile = "Dockerfile"
	}

	if build.Context == "" {
		build.Context = "."
	}

	if build.BaseImage != "" || len(build.ContextDirs) > 0 {
		return buildWithContext(ctx, build)
	}

	return buildImageAndManifest(ctx, build)
}

func buildImageNoManifest(ctx *buildContext, cwd string, build v1.Build) (string, error) {
	_, ids, err := buildkit.Build(ctx.ctx, ctx.pushRepo, true, cwd, nil, build, ctx.messages, ctx.keychain)
	if err != nil {
		return "", err
	}
	return ids[0], nil
}

func buildImageAndManifest(ctx *buildContext, build v1.Build) (string, error) {
	platforms, ids, err := buildkit.Build(ctx.ctx, ctx.pushRepo, false, ctx.cwd, ctx.opts.Platforms, build, ctx.messages, ctx.keychain)
	if err != nil {
		return "", err
	}

	if len(ids) == 1 {
		return ids[0], nil
	}

	return createManifest(ids, platforms, ctx.remoteOpts)
}

func buildWithContext(ctx *buildContext, build v1.Build) (string, error) {
	var (
		baseImage = build.BaseImage
	)

	if baseImage == "" {
		newImage, err := buildImageAndManifest(ctx, build.BaseBuild())
		if err != nil {
			return "", err
		}
		baseImage = newImage
	}

	return buildImageAndManifest(ctx, v1.Build{
		Context:            ".",
		Dockerfile:         "Dockerfile",
		DockerfileContents: toContextCopyDockerFile(baseImage, build.ContextDirs),
	})
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

func getAcornfile(ctx *buildContext, path string) ([]byte, error) {
	msg, cancel := ctx.messages.Recv()
	defer cancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx.ctx, 5*time.Second)
	defer timeoutCancel()

	err := ctx.messages.Send(&buildclient.Message{
		Acornfile: path,
	})
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for acornfile [%s]", path)
		case resp := <-msg:
			if resp.Acornfile == path && resp.Packet != nil {
				return resp.Packet.Data, nil
			}
		}
	}
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
