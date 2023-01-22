package cli

import (
	"io"
	"os"

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
		Project:       c.acorn.Project,
		Kubeconfig:    c.acorn.Kubeconfig,
		KubeconfigEnv: os.Getenv("KUBECONFIG"),
		ContextEnv:    os.Getenv("CONTEXT"),
		NamespaceEnv:  os.Getenv("NAMESPACE"),
		AllProjects:   c.acorn.AllProjects,
	}
}

func (c *CommandClientFactory) CreateDefault() (client.Client, error) {
	return project.Client(c.cmd.Context(), c.Options())
}
