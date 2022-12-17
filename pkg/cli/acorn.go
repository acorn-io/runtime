package cli

import (
	"flag"
	"fmt"
	"os"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

func New() *cobra.Command {
	root := cli.Command(&Acorn{}, cobra.Command{
		Long: "Acorn: Containerized Application Packaging Framework",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	})
	cmdContext := client.CommandContext{
		ClientFactory: &client.CmdClient{},
		StdOut:        os.Stdout,
		StdErr:        os.Stderr,
		StdIn:         nil,
	}
	root.AddCommand(
		NewAll(cmdContext),
		NewApiServer(cmdContext),
		NewApp(cmdContext),
		NewBuild(cmdContext),
		NewBuildServer(cmdContext),
		NewCheck(cmdContext),
		NewContainer(cmdContext),
		NewController(cmdContext),
		NewCredential(cmdContext),
		NewRender(cmdContext),
		NewExec(cmdContext),
		NewImage(cmdContext),
		NewInstall(cmdContext),
		NewUninstall(cmdContext),
		NewInfo(cmdContext),
		NewLogs(cmdContext),
		NewCredentialLogin(true, cmdContext),
		NewCredentialLogout(true, cmdContext),
		NewPull(cmdContext),
		NewPush(cmdContext),
		NewRm(cmdContext),
		NewRun(cmdContext),
		NewUpdate(cmdContext),
		NewSecret(cmdContext),
		NewStart(cmdContext),
		NewStop(cmdContext),
		NewTag(cmdContext),
		NewVolume(cmdContext),
		NewWait(cmdContext),
	)
	root.InitDefaultHelpCmd()
	return root
}

type Acorn struct {
	Kubeconfig    string `usage:"Location of a kubeconfig file"`
	Context       string `usage:"Context to use in the kubeconfig file"`
	Namespace     string `usage:"Namespace to work in" default:"acorn"`
	AllNamespaces bool   `usage:"Namespace to work in" default:"acorn" short:"A"`
	Debug         bool   `usage:"Enable debug logging"`
	DebugLevel    int    `usage:"Debug log level (valid 0-9) (default 7)"`
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
	if a.Debug || a.DebugLevel > 0 {
		logging := flag.NewFlagSet("", flag.PanicOnError)
		klog.InitFlags(logging)

		level := a.DebugLevel
		if level == 0 {
			level = 6
		}
		if level > 7 {
			logrus.SetLevel(logrus.TraceLevel)
		} else {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if err := logging.Parse([]string{
			fmt.Sprintf("-v=%d", level),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (a *Acorn) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
