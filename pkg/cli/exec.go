package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	hclient "github.com/ibuildthecloud/herd/pkg/client"
	"github.com/ibuildthecloud/herd/pkg/client/term"
	"github.com/ibuildthecloud/herd/pkg/streams"
	"github.com/rancher/wrangler-cli"
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
}

func (s *Exec) execApp(ctx context.Context, c hclient.Client, app *hclient.App, args []string) error {
	containers, err := c.ContainerReplicaList(ctx, &hclient.ContainerReplicaListOptions{
		App: app.Name,
	})
	if err != nil {
		return err
	}

	var names []string
	for _, container := range containers {
		names = append(names, container.Name)
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
