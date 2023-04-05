package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/deployargs"
	"github.com/acorn-io/acorn/pkg/dev"
	"github.com/acorn-io/acorn/pkg/rulerequest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	Quiet             bool `usage:"Do not print status" short:"q"`
	out               io.Writer
	client            ClientFactory
}

func (s *Dev) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	// Force install prompt if needed
	_, err = c.Info(cmd.Context())
	if err != nil {
		return err
	}

	cwd := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cwd = args[0]
	}

	image := cwd
	isDir, err := isDirectory(cwd)
	if err != nil {
		return err
	}
	opts, err := s.ToOpts()
	if err != nil {
		return err
	}
	opts = opts.ParseAndTranslate(cmd.Context(), c)

	if image != "." {
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

		opts.DeployArgs = deployParams
	}

	var app *v1.App
	if s.Name != "" {
		app, err = c.AppGet(cmd.Context(), s.Name)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if app != nil && image == "." {
			image = app.Spec.Image
		}
		opts.Name = s.Name
	}
	if app != nil {
		s.Profile = append([]string{"dev?"}, s.Profile...)
		return dev.Dev(cmd.Context(), c, s.File, &dev.Options{
			Args: args,
			Build: client.AcornImageBuildOptions{
				Cwd:      ".",
				Profiles: opts.Profiles,
				Attach:   true,
				ImageID:  image,
				AppName:  app.Name,
			},
			Run:               opts,
			Dangerous:         s.Dangerous,
			BidirectionalSync: s.BidirectionalSync,
		})
	}
	if isDir {
		opts.Name = s.Name
		return dev.Dev(cmd.Context(), c, s.File, &dev.Options{
			Args: args,
			Build: client.AcornImageBuildOptions{
				Cwd:      ".",
				Profiles: opts.Profiles,
			},
			Run:               opts,
			Dangerous:         s.Dangerous,
			BidirectionalSync: s.BidirectionalSync,
		})
	}
	s.Profile = append([]string{"dev?"}, s.Profile...)

	if s.Output != "" {
		app := client.ToApp(c.GetNamespace(), image, &opts)
		return outputApp(s.out, s.Output, app)
	}
	app, err = rulerequest.PromptRun(cmd.Context(), c, s.Dangerous, image, opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)
	err = dev.Dev(cmd.Context(), c, s.File, &dev.Options{
		Args: args,
		Build: client.AcornImageBuildOptions{
			Cwd:      ".",
			Profiles: opts.Profiles,
			ImageID:  app.Spec.Image,
			Attach:   true,
			AppName:  app.Name,
		},
		Run:               opts,
		Dangerous:         s.Dangerous,
		BidirectionalSync: s.BidirectionalSync,
	})
	if err != nil {
		return err
	}
	defer func() {
		go func() { _ = dev.LogLoop(cmd.Context(), c, app, nil) }()
		go func() { _ = dev.AppStatusLoop(cmd.Context(), c, app) }()
		<-cmd.Context().Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.AppStop(ctx, app.Name)
	}()
	return nil
}
