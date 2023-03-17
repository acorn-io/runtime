package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/credentials"
	"github.com/acorn-io/acorn/pkg/hub"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCredentialLogin(root bool, c CommandContext) *cobra.Command {
	cmd := cli.Command(&CredentialLogin{client: c.ClientFactory}, cobra.Command{
		Use:     "login [flags] [SERVER_ADDRESS]",
		Aliases: []string{"add"},
		Example: `
acorn login ghcr.io`,
		SilenceUsage: true,
		Short:        "Add registry credentials",
		Args:         cobra.MaximumNArgs(1),
	})
	if root {
		cmd.Aliases = nil
	}
	return cmd
}

type CredentialLogin struct {
	LocalStorage  bool   `usage:"Store credential on local client for push, pull, and build (not run)" short:"l"`
	SkipChecks    bool   `usage:"Bypass login validation checks"`
	PasswordStdin bool   `usage:"Take the password from stdin"`
	Password      string `usage:"Password" short:"p"`
	Username      string `usage:"Username" short:"u"`
	client        ClientFactory
}

func (a *CredentialLogin) Run(cmd *cobra.Command, args []string) error {
	var (
		client client.Client
	)

	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return err
	}

	var serverAddress string
	if len(args) == 0 && a.Password != "" {
		// HubServer slice length is guaranteed to be >=1 by the ReadCLIConfig method
		serverAddress = cfg.HubServers[0]
	} else if len(args) > 0 {
		serverAddress = args[0]
	}

	if serverAddress == "" {
		return cmd.Help()
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

	isHub, err := hub.IsHub(cfg, serverAddress)
	if err != nil {
		return err
	}

	if isHub {
		user, pass, err := hub.Login(cmd.Context(), a.Password, serverAddress)
		if err != nil {
			return err
		}
		a.Username = user
		a.Password = pass
		a.LocalStorage = true
		a.SkipChecks = true
	} else {
		if err := survey.Ask(q, a); err != nil {
			return err
		}
	}

	if !a.LocalStorage {
		client, err = a.client.CreateDefault()
		if err != nil {
			return err
		}
	}

	store, err := credentials.NewStore(cfg, client)
	if err != nil {
		return err
	}

	err = store.Add(cmd.Context(), credentials.Credential{
		ServerAddress: serverAddress,
		Username:      a.Username,
		Password:      a.Password,
		LocalStorage:  a.LocalStorage,
	}, a.SkipChecks)
	if err != nil {
		return err
	}

	if isHub {
		// reload config, could have changed
		cfg, err = config.ReadCLIConfig()
		if err != nil {
			return err
		}

		var projectSet bool
		def, err := hub.DefaultProject(cmd.Context(), serverAddress, a.Username, a.Password)
		if err != nil {
			return err
		}
		if cfg.CurrentProject == "" && def != "" {
			pterm.Info.Printf("Setting default project to %s\n", def)
			cfg.CurrentProject = def
			if err := cfg.Save(); err != nil {
				return err
			}
			projectSet = true
		}

		if !projectSet {
			if def == "" {
				def = fmt.Sprintf("%s/%s/acorn", serverAddress, a.Username)
			}
			pterm.Info.Printf("Run \"acorn projects %s\" to list available projects\n", serverAddress)
			pterm.Info.Printf("Run \"acorn project use %s\" to set default project\n", def)
		}
	}

	pterm.Success.Printf("Login to %s as %s succeeded\n", serverAddress, a.Username)
	return nil
}
