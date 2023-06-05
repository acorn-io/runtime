package cli

import (
	"fmt"
	"io"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
)

var hideUpdateFlags = []string{"dangerous", "memory", "target-namespace", "secret", "volume", "region", "publish-all",
	"publish", "link", "label", "interval", "env", "compute-class", "annotation"}

func NewUpdate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Update{out: c.StdOut, client: c.ClientFactory}, cobra.Command{
		Use:               "update [flags] APP_NAME [deploy flags]",
		SilenceUsage:      true,
		Short:             "Update a deployed app",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})

	toggleHiddenFlags(cmd, hideUpdateFlags, true)
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Update struct {
	RunArgs
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

	s.RunArgs.Name = name

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
		RunArgs: s.RunArgs,
		Wait:    s.Wait,
		Quiet:   s.Quiet,
		Update:  true,
		out:     s.out,
		client:  s.client,
	}
	return r.Run(cmd, append([]string{s.Image}, args...))
}
