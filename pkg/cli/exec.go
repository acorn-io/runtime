package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/spf13/cobra"
)

func NewExec(c CommandContext) *cobra.Command {
	exec := &Exec{client: c.ClientFactory}
	cmd := cli.Command(exec, cobra.Command{
		Use:               "exec [flags] APP_NAME|CONTAINER_NAME CMD",
		SilenceUsage:      true,
		Short:             "Run a command in a container",
		Long:              "Run a command in a container",
		ValidArgsFunction: newCompletion(c.ClientFactory, onlyAppsWithAcornContainer(exec.Container)).withShouldCompleteOptions(exec.debugImageNoComplete).complete,
	})
	cmd.Flags().SetInterspersed(false)

	// This will produce an error if the container flag doesn't exist or a completion function has already
	// been registered for this flag. Not returning the error since neither of these is likely occur.
	if err := cmd.RegisterFlagCompletionFunc("container", newCompletion(c.ClientFactory, acornContainerCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for -c flag: %v\n", err)
	}

	return cmd
}

type Exec struct {
	Interactive bool   `usage:"Not used" short:"i"`
	TTY         bool   `usage:"Not used" short:"t"`
	DebugImage  string `usage:"Use image as container root for command" short:"d"`
	Container   string `usage:"Name of container to exec into" short:"c"`
	client      ClientFactory
}

func appAndArgs(ctx context.Context, c client.Client, args []string) (string, []string, error) {
	if len(args) > 0 {
		return args[0], args[1:], nil
	}

	apps, err := c.AppList(ctx)
	if err != nil {
		return "", nil, err
	}

	var names []string
	for _, app := range apps {
		if !app.Status.Stopped {
			names = append(names, app.Name)
		}
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

func (s *Exec) filterContainers(containers []apiv1.ContainerReplica) (result []apiv1.ContainerReplica) {
	for _, c := range containers {
		if s.Container == "" {
			result = append(result, c)
		} else if c.Spec.ContainerName == s.Container {
			result = append(result, c)
			break
		} else if c.Spec.ContainerName+"."+c.Spec.SidecarName == s.Container {
			result = append(result, c)
			break
		}
	}
	return result
}

func (s *Exec) execApp(ctx context.Context, c client.Client, app *apiv1.App, args []string) error {
	containers, err := c.ContainerReplicaList(ctx, &client.ContainerReplicaListOptions{
		App: app.Name,
	})
	if err != nil {
		return err
	}

	var (
		displayNames []string
		names        = map[string]string{}
	)

	containers = s.filterContainers(containers)

	for _, container := range containers {
		displayName := fmt.Sprintf("%s (%s %s)", container.Name, container.Status.Columns.State, table.FormatCreated(container.CreationTimestamp))
		displayNames = append(displayNames, displayName)
		names[displayName] = container.Name
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

func (s *Exec) execContainer(ctx context.Context, c client.Client, containerName string, args []string) error {
	tty := term.IsTerminal(os.Stdin) && term.IsTerminal(os.Stdout) && term.IsTerminal(os.Stdout)
	cIO, err := c.ContainerReplicaExec(ctx, containerName, args, tty, &client.ContainerReplicaExecOptions{
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
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	name, args, err := appAndArgs(ctx, c, args)
	if err != nil {
		return err
	}

	app, appErr := c.AppGet(ctx, name)
	if appErr == nil {
		return s.execApp(ctx, c, app, args)
	}
	return s.execContainer(ctx, c, name, args)
}

func (s *Exec) debugImageNoComplete(_ []string) bool {
	return s.DebugImage != ""
}
