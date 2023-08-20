package depot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/depot/depot-go/build"
	"github.com/depot/depot-go/machine"
	cliv1 "github.com/depot/depot-go/proto/depot/cli/v1"
	buildkit "github.com/moby/buildkit/client"
	"github.com/sirupsen/logrus"
)

func Client(ctx context.Context, project, token, image, platform string) (*buildkit.Client, func(error), error) {
	b, err := newBuilder(ctx, project, token, image, platform)
	if err != nil {
		return nil, nil, err
	}

	return b.client, b.Close, nil
}

type builder struct {
	project string
	token   string
	machine *machine.Machine
	build   *build.Build
	client  *buildkit.Client
}

func (b *builder) Close(err error) {
	if b.machine != nil {
		if err := b.machine.Release(); err != nil {
			logrus.Errorf("failed to release machine: %v", err)
		}
		b.machine = nil
	}
	if b.build != nil {
		b.build.Finish(err)
		b.build = nil
	}
	if b.client != nil {
		_ = b.client.Close()
		b.client = nil
	}
}

func newBuilder(ctx context.Context, project, token, image, platform string) (_ *builder, returnErr error) {
	if strings.Contains(platform, "arm64") {
		platform = "arm64"
	} else {
		platform = "amd64"
	}

	req := &cliv1.CreateBuildRequest{
		ProjectId: project,
		Options: []*cliv1.BuildOptions{
			{
				Command: cliv1.Command_COMMAND_BUILD,
				Tags:    []string{image},
			},
		},
	}

	build, err := build.NewBuild(ctx, req, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create depot build: %w", err)
	}
	defer func() {
		if returnErr != nil {
			build.Finish(returnErr)
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if returnErr != nil {
			cancel()
		}
	}()

	buildkitMachine, err := machine.Acquire(ctx, build.ID, build.Token, platform)
	if err != nil {
		return nil, err
	}
	defer func() {
		if returnErr != nil {
			_ = buildkitMachine.Release()
		}
	}()

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer timeoutCancel()

	buildkitClient, err := buildkitMachine.Connect(timeoutCtx)
	if err != nil {
		return nil, err
	}

	return &builder{
		project: project,
		token:   token,
		machine: buildkitMachine,
		build:   &build,
		client:  buildkitClient,
	}, nil
}
