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
