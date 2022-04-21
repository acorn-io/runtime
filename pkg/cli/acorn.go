package cli

import (
	"os"
	"strings"

	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func sheep() string {
	// Artist:  Bob Allison
	return strings.ReplaceAll(`
           __  _
       .-.'  !; !-._  __  _
      (_,         .-:'  !; !-._
    ,'o"(        (_,           )
   (__,-'      ,'o"(            )>
      (       (__,-'            )
       !-'._.--._(             )
          |||  |||!-'._.--._.-'
                     |||  |||   (Artist: Bob Allison)

`, "!", "`")
}

func New() *cobra.Command {
	root := cli.Command(&Acorn{}, cobra.Command{
		Long: "\n   Acorn" + sheep() + "Building cute fluffy apps since 2022.",
		Example: `
# Build and run an app
acorn run --dev .`,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	})
	root.AddCommand(
		NewPush(),
		NewPull(),
		NewAll(),
		NewApp(),
		NewVolume(),
		NewImage(),
		NewTag(),
		NewExec(),
		NewContainer(),
		NewStart(),
		NewStop(),
		NewRm(),
		NewRmi(),
		NewBuild(),
		NewRun(),
		NewLogs(),
		NewDev(),
		NewController(),
	)
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
	return nil
}

func (a *Acorn) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
