package cli

import (
	"io"
	"os"
	"strings"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/spf13/cobra"
)

type CommandContext struct {
	ClientFactory ClientFactory
	StdOut        io.Writer
	StdErr        io.Writer
	StdIn         io.Reader
}

type ClientFactory interface {
	CreateDefault() (client.Client, error)
	Options() project.Options
}

type CommandClientFactory struct {
	cmd   *cobra.Command
	acorn *Acorn
}

func (c *CommandClientFactory) Options() project.Options {
	return project.Options{
		Project:     c.acorn.Project,
		Kubeconfig:  c.acorn.Kubeconfig,
		ContextEnv:  os.Getenv("CONTEXT"),
		AllProjects: c.acorn.AllProjects,
	}
}

func (c *CommandClientFactory) CreateDefault() (client.Client, error) {
	return project.Client(c.cmd.Context(), c.Options())
}

func parseArgGetClient(factory ClientFactory, cmd *cobra.Command, arg string) (client.Client, string, error) {
	parsedProject := ""
	localClientFactory := factory
	opts := factory.Options()
	// project needs to be parsed out of arg before a call to name.ParseReference
	parsedProject, arg = parseProjectOffString(arg)
	if parsedProject != "" {
		localAcorn := &Acorn{
			Kubeconfig:  opts.Kubeconfig,
			Project:     parsedProject,
			AllProjects: opts.AllProjects,
		}
		localClientFactory = &CommandClientFactory{
			cmd:   cmd,
			acorn: localAcorn,
		}
	}
	localClient, err := localClientFactory.CreateDefault()
	if err != nil {
		return nil, "", err
	}
	return localClient, arg, nil
}

func parseProjectOffString(name string) (string, string) {
	if parsedProject, after, found := strings.Cut(name, "::"); found {
		return parsedProject, after
	}
	return "", name
}
