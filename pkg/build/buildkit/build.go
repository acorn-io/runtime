package buildkit

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/containerd/console"
	cplatforms "github.com/containerd/containerd/platforms"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/progress/progressui"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func Build(ctx context.Context, client client.Client, cwd string, platforms []v1.Platform, build v1.Build, streams streams.Streams) ([]v1.Platform, []string, error) {
	dialer, err := client.BuilderDialer(ctx)
	if err != nil {
		return nil, nil, err
	}

	bkc, err := buildkit.New(ctx, "", buildkit.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return dialer(ctx)
	}))
	if err != nil {
		return nil, nil, err
	}
	defer bkc.Close()

	var (
		inPodName      = fmt.Sprintf("127.0.0.1:%d/acorn/%s", system.RegistryPort, client.GetNamespace())
		context        = filepath.Join(cwd, build.Context)
		dockerfilePath = filepath.Dir(filepath.Join(cwd, build.Dockerfile))
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
			LocalDirs: map[string]string{
				"context":    context,
				"dockerfile": dockerfilePath,
			},
			Session: []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
			Exports: []buildkit.ExportEntry{
				{
					Type: buildkit.ExporterImage,
					Attrs: map[string]string{
						"name":           inPodName,
						"name-canonical": "",
						"push":           "true",
					},
				},
			},
		}

		for key, value := range build.BuildArgs {
			options.FrontendAttrs["build-arg:"+key] = value
		}

		res, err := bkc.Solve(ctx, nil, options, progress(ctx, streams))
		if err != nil {
			return nil, nil, err
		}

		inClusterName := fmt.Sprintf("127.0.0.1:5001/acorn/%s@%s", client.GetNamespace(), res.ExporterResponse["containerimage.digest"])
		result = append(result, inClusterName)
	}

	return platforms, result, nil
}

func progress(ctx context.Context, streams streams.Streams) chan *buildkit.SolveStatus {
	var (
		c   console.Console
		err error
	)

	if f, ok := streams.Out.(console.File); ok {
		c, err = console.ConsoleFromFile(f)
		if err != nil {
			c = nil
		}
	}

	ch := make(chan *buildkit.SolveStatus, 1)
	go func() { _, _ = progressui.DisplaySolveStatus(ctx, "", c, streams.Err, ch) }()
	return ch
}
