package cli

import (
	"fmt"
	"io"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/deployargs"
	"github.com/acorn-io/acorn/pkg/rulerequest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewUpdate(out io.Writer) *cobra.Command {
	cmd := cli.Command(&Update{out: out}, cobra.Command{
		Use:          "update [flags] APP_NAME [deploy flags]",
		SilenceUsage: true,
		Short:        "Update a deployed app",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.PersistentFlags().Lookup("dangerous").Hidden = true
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Update struct {
	Image string `json:"image,omitempty"`
	RunArgs

	out io.Writer
}

func (s *Update) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	name := args[0]
	image := s.Image

	if image == "" {
		app, err := c.AppGet(cmd.Context(), name)
		if err != nil {
			return err
		}
		image = app.Spec.Image
	}

	_, flags, err := deployargs.ToFlagsFromImage(cmd.Context(), c, image)
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

	if s.Output != "" {
		app, err := client.ToAppUpdate(cmd.Context(), c, name, &opts)
		if err != nil {
			return err
		}
		return outputApp(s.out, s.Output, app)
	}

	app, err := rulerequest.PromptUpdate(cmd.Context(), c, s.Dangerous, name, opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)
	return nil
}
