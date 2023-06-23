package buildkit

import (
	"context"
	"fmt"
	"path/filepath"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/build/authprovider"
	"github.com/acorn-io/runtime/pkg/buildclient"
	cplatforms "github.com/containerd/containerd/platforms"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/uuid"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func Build(ctx context.Context, pushRepo string, local bool, cwd string, platforms []v1.Platform, build v1.Build, messages buildclient.Messages, keychain authn.Keychain) ([]v1.Platform, []string, error) {
	bkc, err := buildkit.New(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	defer bkc.Close()

	var (
		dockerfileName = filepath.Base(build.Dockerfile)
		result         []string
	)

	if len(platforms) == 0 {
		workers, err := bkc.ListWorkers(ctx)
		if err != nil {
			return nil, nil, err
		}
		if len(workers) == 0 {
			return nil, nil, fmt.Errorf("no workers found on buildkit server")
		}
		if len(workers[0].Platforms) == 0 {
			return nil, nil, fmt.Errorf("no platforms found on workers on buildkit server")
		}
		platforms = []v1.Platform{
			{
				Architecture: workers[0].Platforms[0].Architecture,
				OS:           workers[0].Platforms[0].OS,
				OSVersion:    workers[0].Platforms[0].OSVersion,
				OSFeatures:   workers[0].Platforms[0].OSFeatures,
				Variant:      workers[0].Platforms[0].Variant,
			},
		}
	}

	for _, platform := range platforms {
		options := buildkit.SolveOpt{
			Frontend: "dockerfile.v0",
			FrontendAttrs: map[string]string{
				"target":   build.Target,
				"filename": dockerfileName,
				"platform": cplatforms.Format(ocispecs.Platform(platform)),
			},
			Session: []session.Attachable{authprovider.NewProvider(keychain)},
			Exports: []buildkit.ExportEntry{
				{
					Type: buildkit.ExporterImage,
					Attrs: map[string]string{
						"name":           pushRepo,
						"name-canonical": "",
						"push":           "true",
					},
				},
			},
		}

		if local {
			options.LocalDirs = map[string]string{
				"context":    filepath.Join(cwd, build.Context),
				"dockerfile": filepath.Dir(filepath.Join(cwd, build.Dockerfile)),
			}
		} else {
			options.Session = append(options.Session,
				buildclient.NewFileServer(messages,
					filepath.Join(cwd, build.Context),
					filepath.Join(cwd, build.Dockerfile),
					build.DockerfileContents))
		}

		for key, value := range build.BuildArgs {
			options.FrontendAttrs["build-arg:"+key] = value
		}

		ch, progressDone := progress(messages)
		defer func() { <-progressDone }()

		res, err := bkc.Solve(ctx, nil, options, ch)
		if err != nil {
			return nil, nil, err
		}

		imageName := pushRepo + "@" + res.ExporterResponse["containerimage.digest"]
		result = append(result, imageName)
	}

	return platforms, result, nil
}

func progress(messages buildclient.Messages) (chan *buildkit.SolveStatus, chan struct{}) {
	var (
		done      = make(chan struct{})
		ch        = make(chan *buildkit.SolveStatus, 1)
		sessionid = uuid.New().String()
	)

	go func() {
		for status := range ch {
			_ = messages.Send(&buildclient.Message{
				StatusSessionID: sessionid,
				Status:          status,
			})
		}
		close(done)
	}()

	return ch, done
}
