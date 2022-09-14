package cli

import (
	"os"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	root := cli.Command(&Acorn{}, cobra.Command{
		Long: "Acorn: Containerized Application Packaging Framework",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	})
	root.AddCommand(
		NewAll(),
		NewApiServer(),
		NewApp(),
		NewBuild(),
		NewCheck(),
		NewContainer(),
		NewController(),
		NewCredential(),
		NewDashboard(),
		NewRender(),
		NewExec(),
		NewImage(),
		NewInstall(),
		NewUninstall(),
		NewInfo(),
		NewLogs(),
		NewCredentialLogin(true),
		NewCredentialLogout(true),
		NewPull(),
		NewPush(),
		NewRm(),
		NewRun(os.Stdout),
		NewUpdate(os.Stdout),
		NewSecret(),
		NewStart(),
		NewStop(),
		NewTag(),
		NewVolume(),
		NewWait(),
	)
	root.InitDefaultHelpCmd()
	return root
}

type Acorn struct {
	Kubeconfig    string `usage:"Location of a kubeconfig file"`
	Context       string `usage:"Context to use in the kubeconfig file"`
	Namespace     string `usage:"Namespace to work in" default:"acorn"`
	AllNamespaces bool   `usage:"Namespace to work in" default:"acorn" short:"A"`
}

func setEnv(key, value string) error {
	if value != "" && os.Getenv(key) == "" {
		return os.Setenv(key, value)
	}
	return nil
}

func (a *Acorn) PersistentPre(cmd *cobra.Command, args []string) error {
	if err := setEnv("KUBECONFIG", a.Kubeconfig); err != nil {
		return err
	}
	if err := setEnv("CONTEXT", a.Context); err != nil {
		return err
	}
	if err := setEnv("NAMESPACE", a.Namespace); err != nil {
		return err
	}
	if a.AllNamespaces {
		return os.Setenv("NAMESPACE_ALL", "true")
	}
	if !term.IsTerminal(os.Stdout) || !term.IsTerminal(os.Stderr) || os.Getenv("NO_COLOR") != "" || os.Getenv("NOCOLOR") != "" {
		pterm.DisableStyling()
	}
	return nil
}

func (a *Acorn) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
