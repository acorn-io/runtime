package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
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

func (s *Exec) appAndArgs(ctx context.Context, c hclient.Client, args []string) (string, []string, error) {
	if len(args) > 0 {
		return args[0], args[1:], nil
	}

	apps, err := c.AppList(ctx)
	if err != nil {
		return "", nil, err
	}

	var names []string
	for _, app := range apps {
		names = append(names, app.Name)
	}

	var appName string
	switch len(names) {
	case 0:
		return "", nil, fmt.Errorf("failed to find any apps")
	case 1:
		appName = names[0]
	default:
		err = survey.AskOne(&survey.Select{
			Message: "Choose an app:",
			Options: names,
			Default: names[0],
		}, &appName)
		if err != nil {
			return "", nil, err
		}
	}

	return appName, nil, err
}

func (s *Exec) execApp(ctx context.Context, c hclient.Client, app *apiv1.App, args []string) error {
	containers, err := c.ContainerReplicaList(ctx, &hclient.ContainerReplicaListOptions{
		App: app.Name,
	})
	if err != nil {
		return err
	}

	appRequestedContainerPfx := strings.Join([]string{app.Name, s.Container}, ".")

	var (
		displayNames []string
		names        = map[string]string{}
	)
	for _, container := range containers {
		if strings.HasPrefix(container.Name, appRequestedContainerPfx) {
			displayName := fmt.Sprintf("%s (%s %s)", container.Name, container.Status.Columns.State, table.FormatCreated(container.CreationTimestamp))
			displayNames = append(displayNames, displayName)
			names[displayName] = container.Name
		}
	}

	if len(containers) == 0 {
		return fmt.Errorf("failed to find any containers for app %s", app.Name)
	}

	var choice string
	switch len(displayNames) {
	case 0:
		return fmt.Errorf("failed to find any containers for app %s", app.Name)
	case 1:
		choice = displayNames[0]
	default:
		err = survey.AskOne(&survey.Select{
			Message: "Choose a container:",
			Options: displayNames,
			Default: displayNames[0],
		}, &choice)
		if err != nil {
			return err
		}
	}

	return s.execContainer(ctx, c, names[choice], args)
}

func (s *Exec) execContainer(ctx context.Context, c hclient.Client, containerName string, args []string) error {
	tty := term.IsTerminal(os.Stdin) && term.IsTerminal(os.Stdout) && term.IsTerminal(os.Stdout)
	cIO, err := c.ContainerReplicaExec(ctx, containerName, args, tty, &hclient.ContainerReplicaExecOptions{
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

	name, args, err := s.appAndArgs(ctx, c, args)
	if err != nil {
		return err
	}

	app, appErr := c.AppGet(ctx, name)
	if appErr == nil {
		return s.execApp(ctx, c, app, args)
	}
	return s.execContainer(ctx, c, name, args)
}
