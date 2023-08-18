package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/manager"
	"github.com/acorn-io/runtime/pkg/system"
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
	})
	if root {
		cmd.Aliases = nil
	}
	return cmd
}

type CredentialLogin struct {
	LocalStorage      bool   `usage:"Store credential on local client for push, pull, and build (not run)" short:"l"`
	SkipChecks        bool   `usage:"Bypass login validation checks"`
	SetDefaultContext bool   `usage:"Set default context for project names"`
	PasswordStdin     bool   `usage:"Take the password from stdin"`
	Password          string `usage:"Password" short:"p"`
	Username          string `usage:"Username" short:"u"`
	client            ClientFactory
}

func (a *CredentialLogin) Run(cmd *cobra.Command, args []string) error {
	var (
		client client.Client
	)

	cfg, err := a.client.Options().CLIConfig()
	if err != nil {
		return err
	}

	var serverAddress string
	if len(args) == 0 {
		serverAddress = cfg.GetDefaultAcornServer()
	} else if len(args) > 0 {
		serverAddress = args[0]
	}

	if serverAddress == "" {
		return cmd.Help()
	} else if strings.HasPrefix(serverAddress, "gchr.io") || strings.HasPrefix(serverAddress, "ghrc.io") {
		// protect users from this typo
		regParts := strings.Split(serverAddress, ".io")
		return fmt.Errorf("blocking login attempt to %s.io (did you mean to type ghcr.io?)", regParts[0])
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

	isManager, err := manager.IsManager(cfg, serverAddress)
	if err != nil {
		return err
	}

	if !isManager {
		if err = survey.Ask(q, a); err != nil {
			return err
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

		if err = store.Add(cmd.Context(), apiv1.Credential{
			ServerAddress: serverAddress,
			Username:      a.Username,
			Password:      &a.Password,
			LocalStorage:  a.LocalStorage,
		}, a.SkipChecks); err != nil {
			return err
		}
	} else {
		a.Username, a.Password, err = manager.Login(cmd.Context(), cfg, a.Password, serverAddress)
		if err != nil {
			return err
		}

		var projectSet bool
		def, err := manager.DefaultProject(cmd.Context(), serverAddress, a.Password)
		if err != nil {
			return err
		}

		var cfgModified bool
		if cfg.CurrentProject == "" && def != "" {
			pterm.Info.Printf("Setting default project to %s\n", def)
			cfg.CurrentProject = def
			projectSet = true
			cfgModified = true
		}

		if (cfg.DefaultContext == "" && serverAddress == system.DefaultManagerAddress) || a.SetDefaultContext {
			cfg.DefaultContext = fmt.Sprintf("%s/%s", serverAddress, a.Username)
			cfgModified = true
		}

		if cfgModified {
			if err := cfg.Save(); err != nil {
				return err
			}
		}

		if !projectSet {
			if def == "" {
				if cfg.DefaultContext == serverAddress+"/"+a.Username {
					def = "acorn"
				} else {
					def = fmt.Sprintf("%s/%s/acorn", serverAddress, a.Username)
				}
			}
			pterm.Info.Printf("Run \"acorn projects\" to list available projects\n")
			pterm.Info.Printf("Run \"acorn project use %s\" to set default project\n", def)
		}
	}

	pterm.Success.Printf("Login to %s as %s succeeded\n", serverAddress, a.Username)
	return nil
}
