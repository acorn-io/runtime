package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

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

	isDependent, err := s.isDependentName(cmd.Context(), name)
	if err != nil {
		return err
	} else if isDependent {
		return fmt.Errorf(`acorn update does not support directly updating services or nested Acorns.
Instead, update and rebuild the image for the parent Acorn with references to different Acorn images for any services or nested Acorns you want to update.`)
	}

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

// isDependentName checks whether the app name passed by the user is dependent on a parent app.
// This is the case for nested Acorns and for services. The format is <parent name>.<dependent name>.
func (s *Update) isDependentName(ctx context.Context, name string) (bool, error) {
	appName, dependentName, isDependent := strings.Cut(name, ".")
	if isDependent {
		// try to find the app
		client, err := s.client.CreateDefault()
		if err != nil {
			return false, err
		}
		app, err := client.AppGet(ctx, appName)
		if err != nil {
			return false, err
		}

		// check if the app contains a service that matches the dependentName
		for serviceName := range app.Status.AppSpec.Services {
			if serviceName == dependentName {
				return true, nil
			}
		}
		// check if the app contains a nested Acorn that matches the dependentName
		for acornName := range app.Status.AppSpec.Acorns {
			if acornName == dependentName {
				return true, nil
			}
		}
	}

	return false, nil
}
