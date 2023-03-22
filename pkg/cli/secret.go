package cli

import (
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/strings/slices"
)

func NewSecret(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Secret{client: c.ClientFactory}, cobra.Command{
		Use:     "secret [flags] [SECRET_NAME...]",
		Aliases: []string{"secrets", "s"},
		Example: `
acorn secret`,
		SilenceUsage:      true,
		Short:             "Manage secrets",
		ValidArgsFunction: newCompletion(c.ClientFactory, secretsCompletion).checkProjectPrefix().complete,
	})
	cmd.AddCommand(NewSecretCreate(c))
	cmd.AddCommand(NewSecretDelete(c))
	cmd.AddCommand(NewSecretReveal(c))
	cmd.AddCommand(NewSecretEncrypt(c))
	return cmd
}

type Secret struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

func (a *Secret) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	apps, _ := client.AppList(cmd.Context())

	out := table.NewWriter(tables.Secret, a.Quiet, a.Output)
	out.AddFormatFunc("alias", func(obj apiv1.Secret) string {
		return strings.Join(aliases(&obj, apps), ",")
	})

	if len(args) == 1 {
		secret, err := client.SecretGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(*secret)
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

func aliases(secret *apiv1.Secret, apps []apiv1.App) (result []string) {
	names := sets.NewString()
	for _, app := range apps {
		for _, binding := range app.Spec.Secrets {
			if binding.Secret == secret.Name {
				names.Insert(fmt.Sprintf("%s.%s", app.Name, binding.Target))
			}
		}
	}

	if secret.Labels[labels.AcornSecretGenerated] == "true" {
		names.Insert(fmt.Sprintf("%s.%s", secret.Labels[labels.AcornAppName], secret.Labels[labels.AcornSecretName]))
	}

	return names.List()
}
