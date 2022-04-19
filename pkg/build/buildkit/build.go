package buildkit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/console"
	cplatforms "github.com/containerd/containerd/platforms"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/k8sclient"
	"github.com/ibuildthecloud/herd/pkg/streams"
	"github.com/ibuildthecloud/herd/pkg/system"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/progress/progressui"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func Build(ctx context.Context, cwd, namespace string, platforms []v1.Platform, build v1.Build, streams streams.Streams) (result []string, _ error) {
	c, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}

	port, dialer, err := GetBuildkitDialer(ctx, c)
	if err != nil {
		return nil, err
	}

	inPodName := fmt.Sprintf("127.0.0.1:%d/herd/%s", system.RegistryPort, namespace)
	context := filepath.Join(cwd, build.Context)
	dockerfilePath := filepath.Dir(filepath.Join(cwd, build.Dockerfile))
	dockerfileName := filepath.Base(build.Dockerfile)

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

		for key, value := range build.Args {
			options.FrontendAttrs["build-arg:"+key] = value
		}

		bkc, err := buildkit.New(ctx, "", buildkit.WithContextDialer(dialer))
		if err != nil {
			return nil, err
		}
		defer bkc.Close()

		res, err := bkc.Solve(ctx, nil, options, progress(ctx, streams))
		if err != nil {
			return nil, err
		}

		inClusterName := fmt.Sprintf("127.0.0.1:%d/herd/%s@%s", port, namespace, res.ExporterResponse["containerimage.digest"])
		result = append(result, inClusterName)
	}

	return result, nil
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
	go progressui.DisplaySolveStatus(ctx, "", c, streams.Out, ch)
	return ch
}
