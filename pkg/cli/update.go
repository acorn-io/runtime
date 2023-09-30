package cli

import (
	"fmt"
	"io"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

var hideUpdateFlags = []string{"dangerous", "memory", "secret", "volume", "region", "publish-all",
	"publish", "link", "label", "interval", "env", "compute-class", "annotation"}

func NewUpdate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Update{out: c.StdOut, client: c.ClientFactory}, cobra.Command{
		Use:               "update [flags] ACORN_NAME [deploy flags]",
		SilenceUsage:      true,
		Short:             "Update a deployed Acorn",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
		Example: `
  # Change the image on an Acorn called "my-app"
    acorn update --image <new image> my-app

  # Change the image on an Acorn called "my-app" to the contents of the current directory (which must include an Acornfile)
    acorn update --image . my-app

  # Enable auto-upgrade on an Acorn called "my-app"
    acorn update --auto-upgrade my-app`,
	})

	toggleHiddenFlags(cmd, hideUpdateFlags, true)
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type UpdateArgs struct {
	Region        string   `usage:"Region in which to deploy the app, immutable"`
	File          string   `short:"f" usage:"Name of the build file (default \"DIRECTORY/Acornfile\")"`
	ArgsFile      string   `usage:"Default args to apply to run/update command" default:".args.acorn"`
	Volume        []string `usage:"Bind an existing volume (format existing:vol-name,field=value) (ex: pvc-name:app-data)" short:"v" split:"false"`
	Secret        []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
	Link          []string `usage:"Link external app as a service in the current app (format app-name:container-name)"`
	PublishAll    *bool    `usage:"Publish all (true) or none (false) of the defined ports of application" short:"P"`
	Publish       []string `usage:"Publish port of application (format [public:]private) (ex 81:80)" short:"p"`
	Env           []string `usage:"Environment variables to set on running containers" short:"e"`
	Label         []string `usage:"Add labels to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)" short:"l"`
	Annotation    []string `usage:"Add annotations to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)"`
	Dangerous     bool     `usage:"Automatically approve all privileges requested by the application"`
	Output        string   `usage:"Output API request without creating app (json, yaml)" short:"o"`
	NotifyUpgrade *bool    `usage:"If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it"`
	AutoUpgrade   *bool    `usage:"Enabled automatic upgrades."`
	Interval      string   `usage:"If configured for auto-upgrade, this is the time interval at which to check for new releases (ex: 1h, 5m)"`
	Memory        []string `usage:"Set memory for a workload in the format of workload=memory. Only specify an amount to set all workloads. (ex foo=512Mi or 512Mi)" short:"m"`
	ComputeClass  []string `usage:"Set computeclass for a workload in the format of workload=computeclass. Specify a single computeclass to set all workloads. (ex foo=example-class or example-class)"`
}

type Update struct {
	UpdateArgs
	Image          string `usage:"Acorn image name"`
	ConfirmUpgrade bool   `usage:"When an auto-upgrade app is marked as having an upgrade available, pass this flag to confirm the upgrade. Used in conjunction with --notify-upgrade."`
	Pull           bool   `usage:"Re-pull the app's image, which will cause the app to re-deploy if the image has changed"`
	Wait           *bool  `usage:"Wait for app to become ready before command exiting (default: true)"`
	Quiet          bool   `usage:"Do not print status" short:"q"`
	HelpAdvanced   bool   `usage:"Show verbose help text"`

	out    io.Writer
	client ClientFactory
}

func (s *Update) Run(cmd *cobra.Command, args []string) error {
	if s.HelpAdvanced {
		setAdvancedHelp(cmd, hideUpdateFlags, "")
		return cmd.Help()
	}

	// we can't enforce the one argument requirement at the Cobra level since we have to make --help-advanced possible
	// so enforce the argument requirement here
	if len(args) == 0 {
		return fmt.Errorf("requires at least 1 arg(s), only received 0")
	}

	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	name := args[0]
	args = args[1:]

	if s.ConfirmUpgrade && s.Pull {
		return fmt.Errorf("only --confirm-upgrade or --pull can be set at once")
	}

	if s.ConfirmUpgrade {
		err := c.AppConfirmUpgrade(cmd.Context(), name)
		if err != nil {
			return err
		}
		fmt.Println(name)
		return nil
	}

	if s.Pull {
		err := c.AppPullImage(cmd.Context(), name)
		if err != nil {
			return err
		}
		fmt.Println(name)
		return nil
	}

	r := Run{
		RunArgs: s.getRunArgs(name),
		Wait:    s.Wait,
		Quiet:   s.Quiet,
		Update:  true,
		out:     s.out,
		client:  s.client,
	}
	return r.Run(cmd, append([]string{s.Image}, args...))
}

func (s *Update) getRunArgs(name string) RunArgs {
	return RunArgs{
		Name:       name,
		UpdateArgs: s.UpdateArgs,
	}
}
