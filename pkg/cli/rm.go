package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewRm(c CommandContext) *cobra.Command {
	return cli.Command(&Rm{client: c.ClientFactory}, cobra.Command{
		Use: "rm [flags] ACORN_NAME [ACORN_NAME...]",
		Example: `
acorn rm ACORN_NAME
acorn rm --volumes --secrets ACORN_NAME`,
		SilenceUsage:      true,
		Short:             "Delete an acorn, optionally with it's associated secrets and volumes",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).complete,
		Args:              cobra.MinimumNArgs(1),
	})
}

type Rm struct {
	All           bool `usage:"Delete all associated resources (volumes, secrets)" short:"a"`
	Volumes       bool `usage:"Delete acorn and associated volumes" short:"v"`
	Secrets       bool `usage:"Delete acorn and associated secrets" short:"s"`
	Force         bool `usage:"Do not prompt for delete" short:"f"`
	IgnoreCleanup bool `usage:"Delete acorns without running delete jobs"`
	client        ClientFactory
}

func (a *Rm) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	if a.All {
		a.Volumes = true
		a.Secrets = true
	}

	for _, arg := range args {
		err := removeAcorn(cmd.Context(), c, arg, a.IgnoreCleanup, a.Volumes || a.Secrets)
		if err != nil {
			return err
		}
		if a.Volumes {
			err := removeVolume(arg, c, cmd, a.Force)
			if err != nil {
				return err
			}
		}
		if a.Secrets {
			err := removeSecret(arg, c, cmd, a.Force)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
