package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	hclient "github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/spf13/cobra"
)

func NewExec() *cobra.Command {
	cmd := cli.Command(&Exec{}, cobra.Command{
		Use:          "exec [flags] APP_NAME|CONTAINER_NAME CMD",
		SilenceUsage: true,
		Short:        "Run a command in a container",
		Long:         "Run a command in a container",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Exec struct {
	Interactive bool   `usage:"Not used" short:"i"`
	TTY         bool   `usage:"Not used" short:"t"`
	DebugImage  string `usage:"Use image as container root for command" short:"d"`
	Container   string `usage:"Name of container to exec into" short:"c"`
}

func (s *Exec) execApp(ctx context.Context, c hclient.Client, app *apiv1.App, args []string) error {
	containers, err := c.ContainerReplicaList(ctx, &hclient.ContainerReplicaListOptions{
		App: app.Name,
	})
	if err != nil {
		return err
	}

	appRequestedContainerPfx := strings.Join([]string{app.Name, s.Container}, ".")

	var names []string
	for _, container := range containers {
		if strings.HasPrefix(container.Name, appRequestedContainerPfx) {
			names = append(names, container.Name)
		}
	}

	if len(containers) == 0 {
		return fmt.Errorf("failed to find any containers for app %s", app.Name)
	}

	var containerName string
	switch len(names) {
	case 0:
		return fmt.Errorf("failed to find any containers for app %s", app.Name)
	case 1:
		containerName = names[0]
	default:
		err = survey.AskOne(&survey.Select{
			Message: "Choose a container:",
			Options: names,
			Default: names[0],
		}, &containerName)
		if err != nil {
			return err
		}
	}

	return s.execContainer(ctx, c, containerName, args)
}

func (s *Exec) execContainer(ctx context.Context, c hclient.Client, containerName string, args []string) error {
	cIO, err := c.ContainerReplicaExec(ctx, containerName, args, term.IsTerminal(os.Stdin), &hclient.ContainerReplicaExecOptions{
		DebugImage: s.DebugImage,
	})
	if err != nil {
		return err
	}

	exitCode, err := term.Pipe(cIO, streams.Current())
	if err != nil {
		return err
	}
	os.Exit(exitCode)
	return nil
}

func (s *Exec) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	c, err := hclient.Default()
	if err != nil {
		return err
	}

	name := args[0]

	app, appErr := c.AppGet(ctx, name)
	if appErr == nil {
		return s.execApp(ctx, c, app, args[1:])
	}
	return s.execContainer(ctx, c, name, args[1:])
}
