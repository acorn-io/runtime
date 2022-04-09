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
	root := cli.Command(&Herd{}, cobra.Command{
		Long: "\n   Herd" + sheep() + "Building cute fluffy apps since 2022.",
		Example: `
# Build and run an app
herd run --dev .`,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	})
	root.AddCommand(
		NewApp(),
		NewExec(),
		NewContainer(),
		NewStart(),
		NewStop(),
		NewRm(),
		NewBuild(),
		NewRun(),
		NewLogs(),
		NewDev(),
	)
	return root
}

type Herd struct {
	Kubeconfig    string `usage:"Location of a kubeconfig file"`
	Context       string `usage:"Context to use in the kubeconfig file"`
	Namespace     string `usage:"Namespace to work in" default:"herd"`
	AllNamespaces bool   `usage:"Namespace to work in" default:"herd" short:"A"`
}

func setEnv(key, value string) error {
	if value != "" && os.Getenv(key) == "" {
		return os.Setenv(key, value)
	}
	return nil
}

func (a *Herd) PersistentPre(cmd *cobra.Command, args []string) error {
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

func (a *Herd) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
