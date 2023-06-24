package cli

import (
	"github.com/acorn-io/baaah/pkg/typed"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/spf13/cobra"
)

func NewSecretReveal(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Reveal{client: c.ClientFactory}, cobra.Command{
		Use:     "reveal [flags] [SECRET_NAME...]",
		Aliases: []string{"secrets", "s"},
		Example: `
acorn secret`,
		SilenceUsage:      true,
		Short:             "Manage secrets",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, secretsCompletion).complete,
	})
	return cmd
}

type Reveal struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

type revealEntry struct {
	Name  string
	Type  string
	Key   string
	Value string
}

func (a *Reveal) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter([][]string{
		{"NAME", "Name"},
		{"TYPE", "Type"},
		{"KEY", "Key"},
		{"VALUE", "Value"},
	}, a.Quiet, a.Output)

	var matchedSecrets []apiv1.Secret

	for _, arg := range args {
		secret, err := client.SecretReveal(cmd.Context(), arg)
		if err != nil {
			return err
		}
		matchedSecrets = append(matchedSecrets, *secret)
	}

	for _, secret := range matchedSecrets {
		for _, entry := range typed.Sorted(secret.Data) {
			out.WriteFormatted(&revealEntry{
				Name:  secret.Name,
				Type:  secret.Type,
				Key:   entry.Key,
				Value: string(entry.Value),
			}, &secret)
			if a.Output != "" {
				// in non-table output, we write the source object to buffer,
				// so we exit here to not write the same object multiple times (one per data key)
				break
			}
		}
	}

	return out.Err()
}
