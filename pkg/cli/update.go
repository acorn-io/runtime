package cli

import (
	"fmt"
	"io"

	"github.com/acorn-io/acorn/pkg/autoupgrade"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/deployargs"
	"github.com/acorn-io/acorn/pkg/rulerequest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewUpdate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Update{out: c.StdOut, client: c.ClientFactory}, cobra.Command{
		Use:               "update [flags] APP_NAME [deploy flags]",
		SilenceUsage:      true,
		Short:             "Update a deployed app",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})
	cmd.PersistentFlags().Lookup("dangerous").Hidden = true
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Update struct {
	Image          string `json:"image,omitempty"`
	Replace        bool   `usage:"Toggle replacing update, resetting undefined fields to default values" json:"replace,omitempty"` // Replace sets patchMode to false, resulting in a full update, resetting all undefined fields to their defaults
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

	name := args[0]
	image := s.Image

	if s.ConfirmUpgrade {
		if image != "" {
			return fmt.Errorf("cannot set an image (%v) and confirm ann upgrade at the same time", image)
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

	imageForFlags := image
	if imageForFlags == "" {
		imageForFlags = app.Spec.Image

		if _, isPattern := autoupgrade.AutoUpgradePattern(imageForFlags); isPattern {
			imageForFlags = app.Status.AppImage.ID
		}
	}

	if imageForFlags == "" {
		return fmt.Errorf("cannot update app. Image not found")
	}

	_, flags, err := deployargs.ToFlagsFromImage(cmd.Context(), c, imageForFlags)
	if err != nil {
		return err
	}

	deployParams, err := flags.Parse(args)
	if pflag.ErrHelp == err {
		return nil
	} else if err != nil {
		return err
	}

	runOpts, err := s.ToOpts()
	if err != nil {
		return err
	}

	opts := runOpts.ToUpdate()
	opts.Image = image
	opts.DeployArgs = deployParams

	// Overwrite == true means patchMode == false
	opts.Replace = s.Replace

	if s.Output != "" {
		app, err := client.ToAppUpdate(cmd.Context(), c, name, &opts)
		if err != nil {
			return err
		}
		return outputApp(s.out, s.Output, app)
	}

	app, err = rulerequest.PromptUpdate(cmd.Context(), c, s.Dangerous, name, opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)
	return nil
}
