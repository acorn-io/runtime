package cli

import (
	"io"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/dev"
	"github.com/acorn-io/acorn/pkg/imagesource"
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
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Dev struct {
	RunArgs
	BidirectionalSync bool `usage:"In interactive mode download changes in addition to uploading" short:"b"`
	Replace           bool `usage:"Replace the app with only defined values, resetting undefined fields to default values" json:"replace,omitempty"` // Replace sets patchMode to false, resulting in a full update, resetting all undefined fields to their defaults
	out               io.Writer
	client            ClientFactory
}

func (s *Dev) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	imageSource := imagesource.NewImageSource(s.File, args, s.Profile, nil)

	opts, err := s.ToOpts(cmd.Context(), c)
	if err != nil {
		return err
	}

	return dev.Dev(cmd.Context(), c, &dev.Options{
		ImageSource:       imageSource,
		Run:               opts,
		Replace:           s.Replace,
		Dangerous:         s.Dangerous,
		BidirectionalSync: s.BidirectionalSync,
	})
}
