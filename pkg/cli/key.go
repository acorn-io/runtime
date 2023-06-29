package cli

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
)

func NewKey(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Key{client: c.ClientFactory}, cobra.Command{
		Use:     "key [flags]",
		Aliases: []string{"keys"},
		Example: `
acorn key`,
		SilenceUsage: true,
		Short:        "Manage (public) keys",
		Args:         cobra.MaximumNArgs(1),
	})
	cmd.AddCommand(NewKeyImport(c))
	return cmd
}

type Key struct {
	client ClientFactory
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o" local:"true"`
}

func (a *Key) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	var keys []apiv1.PublicKey

	if len(args) != 0 {
		key, err := client.KeyGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		keys = append(keys, *key)
	} else {
		keys, err = client.KeyList(cmd.Context())
		if err != nil {
			return err
		}
	}

	out := table.NewWriter(tables.PublicKey, false, a.Output)
	for _, key := range keys {
		out.Write(key)
	}

	return out.Err()
}
