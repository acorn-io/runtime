package cli

import (
	"fmt"
	"io"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewUpdate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Update{out: c.StdOut, client: c.ClientFactory}, cobra.Command{
		Use:               "update [flags] APP_NAME [deploy flags]",
		SilenceUsage:      true,
		Short:             "Update a deployed app",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
		Args:              cobra.MinimumNArgs(1),
	})
	hideUpdateFlags := []string{"dangerous", "memory", "target-namespace", "secret", "volume", "region", "publish-all",
		"publish", "link", "label", "interval", "image", "env", "compute-class", "annotation"}

	toggleHiddenFlags(cmd, hideUpdateFlags, true)
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Update struct {
	RunArgs
	Image          string `usage:"Acorn image name"`
	ConfirmUpgrade bool   `usage:"When an auto-upgrade app is marked as having an upgrade available, pass this flag to confirm the upgrade. Used in conjunction with --notify-upgrade."`
	Pull           bool   `usage:"Re-pull the app's image, which will cause the app to re-deploy if the image has changed"`
	Replace        bool   `usage:"Replace the app with only defined values, resetting undefined fields to default values" json:"replace,omitempty"` // Replace sets patchMode to false, resulting in a full update, resetting all undefined fields to their defaults
	Wait           *bool  `usage:"Wait for app to become ready before command exiting (default: true)"`
	Quiet          bool   `usage:"Do not print status" short:"q"`

	out    io.Writer
	client ClientFactory
}

func (s *Update) Run(cmd *cobra.Command, args []string) error {
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
		Replace: s.Replace,
		out:     s.out,
		client:  s.client,
	}
	return r.Run(cmd, append([]string{s.Image}, args...))
}
