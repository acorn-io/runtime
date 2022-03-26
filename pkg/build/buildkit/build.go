package buildkit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/console"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/k8sclient"
	"github.com/ibuildthecloud/herd/pkg/streams"
	"github.com/ibuildthecloud/herd/pkg/system"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/progress/progressui"
)

func Build(ctx context.Context, cwd string, build v1.Build, streams streams.Streams) (string, error) {
	c, err := k8sclient.Default()
	if err != nil {
		return "", err
	}

	port, dialer, err := GetBuildkitDialer(ctx, c)
	if err != nil {
		return "", err
	}

	uuid := build.Hash()[:12]
	inPodName := fmt.Sprintf("127.0.0.1:%d/%s", system.RegistryPort, uuid)
	context := filepath.Join(cwd, build.Context)
	dockerfilePath := filepath.Dir(filepath.Join(cwd, build.Dockerfile))
	dockerfileName := filepath.Base(build.Dockerfile)

	options := buildkit.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"target":   build.Target,
			"filename": dockerfileName,
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

	bkc, err := buildkit.New(ctx, "", buildkit.WithContextDialer(dialer))
	if err != nil {
		return "", err
	}
	defer bkc.Close()

	res, err := bkc.Solve(ctx, nil, options, progress(ctx, streams))
	if err != nil {
		return "", err
	}

	inClusterName := fmt.Sprintf("127.0.0.1:%d/%s@%s", port, uuid, res.ExporterResponse["containerimage.digest"])
	return inClusterName, nil
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
