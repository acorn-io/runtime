package cli

import (
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"strings"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/spf13/cobra"
)

func NewRm(c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&Rm{client: c.ClientFactory}, cobra.Command{
		Use: "rm [flags] [APP_NAME...]",
		Example: `
acorn rm APP_NAME
acorn rm -t volume,container APP_NAME`,
		SilenceUsage: true,
		Short:        "Delete an app, container, secret or volume",
	})
	return cmd
}

type Rm struct {
	All    bool     `usage:"Delete all types" short:"a"`
	Type   []string `usage:"Delete by type (container,app,volume,secret or c,a,v,s)" short:"t"`
	Force  bool     `usage:"Force Delete" short:"f"`
	client client.ClientFactory
}
type RmObjects struct {
	App       bool
	Container bool
	Secret    bool
	Volume    bool
}

func (a *Rm) Run(cmd *cobra.Command, args []string) error {
	var rmObjects RmObjects
	cfg, err := restconfig.Default()
	if err != nil {
		return err
	}

	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		pterm.Error.Println("No AppName arg provided")
		return errors.New("No AppName arg provided")
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

	for _, arg := range args {
		c := c
		ns, name, ok := strings.Cut(arg, "/")
		if ok {
			c, err = client.New(cfg, ns)
			if err != nil {
				return err
			}
			arg = name
		}
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
