package cli

import (
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler-cli/pkg/table"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewCredential() *cobra.Command {
	cmd := cli.Command(&Credential{}, cobra.Command{
		Use:     "credential [flags] [SERVER_ADDRESS...]",
		Aliases: []string{"credentials", "creds"},
		Example: `
acorn credential`,
		SilenceUsage: true,
		Short:        "Manage registry credentials",
	})
	cmd.AddCommand(NewCredentialLogin())
	cmd.AddCommand(NewCredentialLogout())
	return cmd
}

type Credential struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

func (a *Credential) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Credential, "", a.Quiet, a.Output)

	if len(args) == 1 {
		credential, err := client.CredentialGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(credential)
		return out.Err()
	}

	credentials, err := client.CredentialList(cmd.Context())
	if err != nil {
		return err
	}

	for _, credential := range credentials {
		if len(args) > 0 {
			if slices.Contains(args, credential.Name) {
				out.Write(credential)
			}
		} else {
			out.Write(credential)
		}
	}

	return out.Err()
}
