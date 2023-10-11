package cli

import (
	"io"

	"github.com/acorn-io/runtime/pkg/autoupgrade"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/dev"
	"github.com/acorn-io/runtime/pkg/imagesource"
	"github.com/acorn-io/runtime/pkg/vcs"
	"github.com/acorn-io/z"
	"github.com/spf13/cobra"
)

func NewDev(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Dev{out: c.StdOut, client: c.ClientFactory}, cobra.Command{
		Use:               "dev [flags] IMAGE|DIRECTORY [acorn args]",
		SilenceUsage:      true,
		Short:             "Run an app from an image or Acornfile in dev mode or attach a dev session to a currently running app",
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).withSuccessDirective(cobra.ShellCompDirectiveDefault).withShouldCompleteOptions(onlyNumArgs(1)).complete,
		Example: `
acorn dev <IMAGE>
acorn dev .
acorn dev --name wandering-sound
acorn dev --name wandering-sound <IMAGE>
`})

	// This will produce an error if the volume flag doesn't exist or a completion function has already
	// been registered for this flag. Not returning the error since neither of these is likely occur.
	if err := cmd.RegisterFlagCompletionFunc("volume", newCompletion(c.ClientFactory, volumeFlagClassCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for -v flag: %v\n", err)
	}
	// This will produce an error if the computeclass flag doesn't exist or a completion function has already
	// been registered for this flag. Not returning the error since neither of these is likely occur.
	if err := cmd.RegisterFlagCompletionFunc("compute-class", newCompletion(c.ClientFactory, computeClassFlagCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for --compute-class flag: %v\n", err)
	}
	cmd.PersistentFlags().Lookup("dangerous").Hidden = true
	toggleHiddenFlags(cmd, hideUpdateFlags, true)
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Dev struct {
	RunArgs
	BidirectionalSync bool   `usage:"In interactive mode download changes in addition to uploading" short:"b"`
	Replace           bool   `usage:"Replace the app with only defined values, resetting undefined fields to default values" json:"replace,omitempty"` // Replace sets patchMode to false, resulting in a full update, resetting all undefined fields to their defaults
	Clone             string `usage:"Clone a running app"`
	HelpAdvanced      bool   `usage:"Show verbose help text"`
	out               io.Writer
	client            ClientFactory
}

func (s *Dev) Run(cmd *cobra.Command, args []string) error {
	if s.HelpAdvanced {
		setAdvancedHelp(cmd, hideUpdateFlags, AdvancedHelp)
		return cmd.Help()
	}
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	var imageSource imagesource.ImageSource
	if s.Clone == "" {
		imageSource = imagesource.NewImageSource(s.client.AcornConfigFile(), s.File, s.ArgsFile, args, nil, z.Dereference(s.AutoUpgrade))
	} else {
		// Get info from the running app
		app, err := c.AppGet(cmd.Context(), s.Clone)
		if err != nil {
			return err
		}

		acornfile, err := vcs.AcornfileFromApp(cmd.Context(), app)
		if err != nil {
			return err
		}

		imageSource = imagesource.NewImageSource(s.client.AcornConfigFile(), acornfile, s.ArgsFile, args, nil, z.Dereference(s.AutoUpgrade))
	}

	opts, err := s.ToOpts()
	if err != nil {
		return err
	}

	// If auto-upgrade is not set, set it to true if auto-upgrade is implied
	if !z.Dereference(opts.AutoUpgrade) && autoupgrade.Implied(imageSource.Image, s.Interval, z.Dereference(opts.NotifyUpgrade)) {
		opts.AutoUpgrade = z.Pointer(true)
	}

	return dev.Dev(cmd.Context(), c, &dev.Options{
		ImageSource:       imageSource,
		Run:               opts,
		Replace:           s.Replace,
		Dangerous:         s.Dangerous,
		BidirectionalSync: s.BidirectionalSync,
	})
}
