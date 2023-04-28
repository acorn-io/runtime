package cli

import (
	"fmt"
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

// parseProjectOffString parses a project off of a string of the potential form project::resource
// and returns the project (or "" if none is found) and the resource
func parseProjectOffString(name string) (string, string) {
	if parsedProject, parsedResource, found := strings.Cut(name, "::"); found {
		return parsedProject, parsedResource
	}
	return "", name
}

// noCrossProjectArgs returns an error if any of the args are specifying a different project
// and converts any args of project::resource form to resource form if it is of the current project
func noCrossProjectArgs(args []string, currentProject string) ([]string, error) {
	var parsedProject string
	for i := range args {
		parsedProject, args[i] = parseProjectOffString(args[i])
		if len(parsedProject) > 0 && parsedProject != currentProject {
			return nil, fmt.Errorf("cannot cross project boundaries with acorn run")
		}
	}
	return args, nil
}
