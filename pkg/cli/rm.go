package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewRm(c CommandContext) *cobra.Command {
	return cli.Command(&Rm{client: c.ClientFactory}, cobra.Command{
		Use: "rm [flags] APP_NAME [APP_NAME...]",
		Example: `
acorn rm APP_NAME
acorn rm -t volume,container APP_NAME`,
		SilenceUsage:      true,
		Short:             "Delete an app, container, secret or volume",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).complete,
		Args:              cobra.MinimumNArgs(1),
	})
}

type Rm struct {
	All    bool     `usage:"Delete all types" short:"a"`
	Type   []string `usage:"Delete by type (container,app,volume,secret or c,a,v,s)" short:"t"`
	Force  bool     `usage:"Force Delete" short:"f"`
	client ClientFactory
}
type RmObjects struct {
	App       bool
	Container bool
	Secret    bool
	Volume    bool
}

func (a *Rm) Run(cmd *cobra.Command, args []string) error {
	var rmObjects RmObjects

	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	if a.All {
		rmObjects = RmObjects{
			App:    true,
			Secret: true,
			Volume: true,
		}
	} else if len(a.Type) > 0 {
		for _, obj := range a.Type {
			addRmObject(&rmObjects, obj)
		}
	} else { // If nothing is set default to App
		rmObjects = RmObjects{
			App: true,
		}
	}
	// Do not prompt when deleting non-nested resource
	a.Force = a.Force || rmObjects.App && !rmObjects.Secret && !rmObjects.Volume

	for _, arg := range args {
		if rmObjects.App {
			err := removeApp(arg, c, cmd, a.Force)
			if err != nil {
				return err
			}
		}
		if rmObjects.Container {
			err := removeContainer(arg, c, cmd, a.Force)
			if err != nil {
				return err
			}
		}
		if rmObjects.Volume {
			err := removeVolume(arg, c, cmd, a.Force)
			if err != nil {
				return err
			}
		}
		if rmObjects.Secret {
			err := removeSecret(arg, c, cmd, a.Force)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
