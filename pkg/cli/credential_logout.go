package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/config"
	credentials2 "github.com/acorn-io/acorn/pkg/credentials"
	"github.com/spf13/cobra"
)

func NewCredentialLogout(root bool, c CommandContext) *cobra.Command {
	cmd := cli.Command(&CredentialLogout{client: c.ClientFactory}, cobra.Command{
		Use:     "logout [flags] [SERVER_ADDRESS]",
		Aliases: []string{"rm"},
		Example: `
acorn logout ghcr.io`,
		SilenceUsage:      true,
		Short:             "Remove registry credentials",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, credentialsCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})
	if root {
		cmd.Aliases = nil
	}
	return cmd
}

type CredentialLogout struct {
	client       ClientFactory
	LocalStorage bool `usage:"Delete locally stored credential (not remotely stored)" short:"l"`
}

func (a *CredentialLogout) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return err
	}

	store, err := credentials2.NewStore(cfg, client)
	if err != nil {
		return err
	}

	err = store.Remove(cmd.Context(), credentials2.Credential{
		ServerAddress: args[0],
		LocalStorage:  a.LocalStorage,
	})
	if err != nil {
		return err
	}

	// reload config
	cfg, err = config.ReadCLIConfig()
	if err != nil {
		return fmt.Errorf("failed to remove server %s from CLI config: %v", args[0], err)
	}

	return config.RemoveServer(cfg, args[0])
}
