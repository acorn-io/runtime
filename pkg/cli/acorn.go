package cli

import (
	"os"

	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	root := cli.Command(&Acorn{}, cobra.Command{
		Long: "Acorn: Portable Kubernetes Applications",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	})
	root.AddCommand(
		NewPush(),
		NewPull(),
		NewAll(),
		NewApiServer(),
		NewApp(),
		NewVolume(),
		NewImage(),
		NewInit(),
		NewTag(),
		NewExec(),
		NewContainer(),
		NewStart(),
		NewStop(),
		NewRm(),
		NewBuild(),
		NewRun(),
		NewLogs(),
		NewDev(),
		NewController(),
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
	return nil
}

func (a *Acorn) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
