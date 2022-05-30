package cli

import (
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler-cli/pkg/table"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewSecret() *cobra.Command {
	cmd := cli.Command(&Secret{}, cobra.Command{
		Use:     "secret [flags] [SECRET_NAME...]",
		Aliases: []string{"secrets", "s"},
		Example: `
acorn secret`,
		SilenceUsage: true,
		Short:        "Manage secrets",
	})
	cmd.AddCommand(NewSecretCreate())
	cmd.AddCommand(NewSecretDelete())
	cmd.AddCommand(NewSecretExpose())
	return cmd
}

type Secret struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

func (a *Secret) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Secret, "", a.Quiet, a.Output)

	if len(args) == 1 {
		secret, err := client.SecretGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(secret)
		return out.Err()
	}

	secrets, err := client.SecretList(cmd.Context())
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if len(args) > 0 {
			if slices.Contains(args, secret.Name) {
				out.Write(secret)
			}
		} else {
			out.Write(secret)
		}
	}

	return out.Err()
}
