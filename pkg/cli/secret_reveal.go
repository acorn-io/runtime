package cli

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/spf13/cobra"
)

func NewSecretReveal(c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&Reveal{client: c.ClientFactory}, cobra.Command{
		Use:          "reveal [flags] [SECRET_NAME...]",
		Example:      `acorn secret reveal foo-secret-ab123`,
		SilenceUsage: true,
		Short:        "Reveal the values of a secret.",
		Args:         cobra.MinimumNArgs(1),
	})
	return cmd
}

type Reveal struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client client.ClientFactory
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
		{"Name", "Name"},
		{"Type", "Type"},
		{"Key", "Key"},
		{"Value", "Value"},
	}, system.UserNamespace(), a.Quiet, a.Output)

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
			out.Write(&revealEntry{
				Name:  secret.Name,
				Type:  secret.Type,
				Key:   entry.Key,
				Value: string(entry.Value),
			})
		}
	}

	return out.Err()
}
