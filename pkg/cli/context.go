package cli

import (
	"io"
	"os"

	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/project"
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
	CreateWithAllProjects() (client.Client, error)
	CreateWithAllProjectsAllServers() (client.Client, error)
	Options() project.Options
	AcornConfigFile() string
}

type CommandClientFactory struct {
	cmd   *cobra.Command
	acorn *Acorn
}

func (c *CommandClientFactory) Options() project.Options {
	return project.Options{
		AcornConfigFile: c.acorn.AcornConfigFile,
		Project:         c.acorn.Project,
		Kubeconfig:      c.acorn.Kubeconfig,
		ContextEnv:      os.Getenv("CONTEXT"),
	}
}

func (c *CommandClientFactory) CreateDefault() (client.Client, error) {
	return project.Client(c.cmd.Context(), c.Options())
}

func (c *CommandClientFactory) CreateWithAllProjects() (client.Client, error) {
	opts := c.Options()
	opts.AllProjects = true
	return project.Client(c.cmd.Context(), opts)
}

func (c *CommandClientFactory) CreateWithAllProjectsAllServers() (client.Client, error) {
	opts := c.Options()
	opts.AllProjects = true
	opts.AllServers = true
	return project.Client(c.cmd.Context(), opts)
}

func (c *CommandClientFactory) AcornConfigFile() string {
	return c.acorn.AcornConfigFile
}
