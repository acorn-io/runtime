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
	})
	cmd.PersistentFlags().Lookup("dangerous").Hidden = true
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Update struct {
	Image          string `json:"image,omitempty"`
	ConfirmUpgrade bool   `usage:"When an auto-upgrade app is marked as having an upgrade available, pass this flag to confirm the upgrade. Used in conjunction with --notify-upgrade."`
	Pull           bool   `usage:"Re-pull the app's image, which will cause the app to re-deploy if the image has changed"`
	RunArgs

	out    io.Writer
	client ClientFactory
}

func (s *Update) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	image := s.Image
	if !s.ConfirmUpgrade && !s.Pull {
		r := Run{
			RunArgs:     s.RunArgs,
			Interactive: false,
			Update:      true,
			Image:       s.Image,
			out:         s.out,
			client:      s.client,
		}
		err := r.Run(cmd, append([]string{}, args...))
		return err
	}
	if s.ConfirmUpgrade {
		if image != "" {
			return fmt.Errorf("cannot set an image (%v) and confirm an upgrade at the same time", image)
		}

		err := c.AppConfirmUpgrade(cmd.Context(), name)
		if err != nil {
			return err
		}
	}

	app, err := c.AppGet(cmd.Context(), name)
	if err != nil {
		return err
	}

	if s.Pull || image == app.Spec.Image {
		if s.Pull && image != "" && image != app.Spec.Image {
			return fmt.Errorf("cannot change image (%v) and specify --pull at the same time", image)
		}

		err := c.AppPullImage(cmd.Context(), name)
		if err != nil {
			return err
		}
	}
	return nil
}
