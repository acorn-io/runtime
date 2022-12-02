package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func NewCredentialLogin(root bool, c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&CredentialLogin{client: c.ClientFactory}, cobra.Command{
		Use:     "login [flags] [SERVER_ADDRESS]",
		Aliases: []string{"add"},
		Example: `
acorn login ghcr.io`,
		SilenceUsage: true,
		Short:        "Add registry credentials",
		Args:         cobra.ExactArgs(1),
	})
	if root {
		cmd.Aliases = nil
	}
	return cmd
}

type CredentialLogin struct {
	SkipChecks    bool   `usage:"Bypass login validation checks"`
	PasswordStdin bool   `usage:"Take the password from stdin"`
	Password      string `usage:"Password" short:"p"`
	Username      string `usage:"Username" short:"u"`
	client        client.ClientFactory
}

func (a *CredentialLogin) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	if a.PasswordStdin {
		contents, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		a.Password = strings.TrimSuffix(string(contents), "\n")
		a.Password = strings.TrimSuffix(a.Password, "\r")
	}

	var q []*survey.Question
	if a.Username == "" {
		q = append(q, &survey.Question{
			Name:   "username",
			Prompt: &survey.Input{Message: "Username"},
		})
	}
	if a.Password == "" {
		q = append(q, &survey.Question{
			Name:   "password",
			Prompt: &survey.Password{Message: "Password"},
		})
	}

	if err := survey.Ask(q, a); err != nil {
		return err
	}

	existing, err := client.CredentialGet(cmd.Context(), args[0])
	if apierror.IsNotFound(err) {
		cred, err := client.CredentialCreate(cmd.Context(), args[0], a.Username, a.Password, a.SkipChecks)
		if err != nil {
			return err
		}

		fmt.Println(cred.Name)
		return nil
	}

	existing.Username = a.Username
	existing.Password = &a.Password
	cred, err := client.CredentialUpdate(cmd.Context(), args[0], a.Username, a.Password, a.SkipChecks)
	if err != nil {
		return err
	}
	fmt.Println(cred.Name)
	return nil
}
