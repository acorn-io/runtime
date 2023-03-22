package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

func New() *cobra.Command {
	a := &Acorn{}
	root := cli.Command(a, cobra.Command{
		Long: "Acorn: Containerized Application Packaging Framework",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	})
	cmdContext := CommandContext{
		ClientFactory: &CommandClientFactory{
			cmd:   root,
			acorn: a,
		},
		StdOut: os.Stdout,
		StdErr: os.Stderr,
		StdIn:  nil,
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
		NewFmt(cmdContext),
		NewImage(cmdContext),
		NewInstall(cmdContext),
		NewOfferings(cmdContext),
		NewUninstall(cmdContext),
		NewInfo(cmdContext),
		NewLogs(cmdContext),
		NewCredentialLogin(true, cmdContext),
		NewCredentialLogout(true, cmdContext),
		NewProject(cmdContext),
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
	// This will produce an error if the project flag doesn't exist or a completion function has already
	// been registered for this flag. Not returning the error since neither of these is likely occur.
	if err := root.RegisterFlagCompletionFunc("project", newCompletion(cmdContext.ClientFactory, projectsCompletion).complete); err != nil {
		root.Printf("Error registering completion function for -j flag: %v\n", err)
	}
	root.InitDefaultHelpCmd()
	return root
}

type Acorn struct {
	Kubeconfig  string `usage:"Explicitly use kubeconfig file, overriding current project"`
	Project     string `usage:"Project to work in" short:"j" env:"ACORN_PROJECT"`
	AllProjects bool   `usage:"Use all known projects" short:"A" env:"ACORN_ALL_PROJECTS"`
	Debug       bool   `usage:"Enable debug logging" env:"ACORN_DEBUG"`
	DebugLevel  int    `usage:"Debug log level (valid 0-9) (default 7)" env:"ACORN_DEBUG_LEVEL"`
}

func setEnv(key, value string) error {
	if value != "" && os.Getenv(key) == "" {
		return os.Setenv(key, value)
	}
	return nil
}

func (a *Acorn) PersistentPre(cmd *cobra.Command, args []string) error {
	// If --kubeconfig is used set it to KUBECONFIG env (if env is unset) so that all
	// kubeconfig file looks will find it
	if err := setEnv("KUBECONFIG", a.Kubeconfig); err != nil {
		return err
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
			logs.Debug = log.New(os.Stderr, "ggcr: ", log.LstdFlags)
		} else {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if err := logging.Parse([]string{
			fmt.Sprintf("-v=%d", level),
		}); err != nil {
			return err
		}
	}

	if a.Project != "" {
		clientFactory := CommandClientFactory{
			cmd:   cmd,
			acorn: a,
		}
		cfg, err := config.ReadCLIConfig()
		if err != nil {
			return err
		}

		err = project.Exists(cmd.Context(), clientFactory.Options().WithCLIConfig(cfg), a.Project)
		if err != nil {
			return err
		}
	}

	// parse the form project::resource
	for i, arg := range args {
		if projectOverride, newArg, ok := strings.Cut(arg, "::"); ok {
			args[i] = newArg
			a.Project = projectOverride
		}
	}
	cmd.SetArgs(args)

	return nil
}

// This runs from running acorn. How do I get this to run before all commands
//func (a *Acorn) Pre(cmd *cobra.Command, args []string) error {
//
//
//}

func (a *Acorn) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
