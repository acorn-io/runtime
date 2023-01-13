package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewCredential(c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&Credential{client: c.ClientFactory}, cobra.Command{
		Use:     "credential [flags] [SERVER_ADDRESS...]",
		Aliases: []string{"credentials", "creds"},
		Example: `
acorn credential`,
		SilenceUsage:      true,
		Short:             "Manage registry credentials",
		ValidArgsFunction: newCompletion(c.ClientFactory, credentialsCompletion).complete,
	})
	cmd.AddCommand(NewCredentialLogin(false, c))
	cmd.AddCommand(NewCredentialLogout(false, c))
	return cmd
}

type Credential struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client client.ClientFactory
}

func (a *Credential) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Credential, system.UserNamespace(), a.Quiet, a.Output)

	if len(args) == 1 {
		credential, err := c.CredentialGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(credential)
		return out.Err()
	}

	credentials, err := c.CredentialList(cmd.Context())
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
