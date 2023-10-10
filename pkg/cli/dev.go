package cli

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/acorn-io/runtime/pkg/autoupgrade"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/dev"
	"github.com/acorn-io/runtime/pkg/imagesource"
	"github.com/acorn-io/z"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
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
		vcs := app.Status.Staged.AppImage.VCS

		if len(vcs.Remotes) == 0 {
			return fmt.Errorf("clone can only be done on an app built from a git repository")
		}

		for _, remote := range vcs.Remotes {
			var gitUrl string
			httpUrl, err := url.Parse(remote)
			if err == nil {
				gitUrl = fmt.Sprintf("git@%s:%s", httpUrl.Host, httpUrl.Path[1:])
			} else {
				gitUrl = remote
			}

			// TODO workdir named after git repo, cloned app name, or just this app's name?
			idx := strings.LastIndex(gitUrl, "/")
			if idx < 0 || idx >= len(gitUrl) {
				fmt.Printf("failed to determine repository name %q\n", gitUrl)
				continue
			}
			workdir := strings.TrimSuffix(gitUrl[idx+1:], ".git")

			// Clone git repo
			auth, _ := ssh.NewSSHAgentAuth("git")
			_, err = git.PlainCloneContext(cmd.Context(), workdir, false, &git.CloneOptions{
				URL: gitUrl,
				// TODO print progress to somewhere maybe
				// Progress: os.Stderr/os.Stdout,
				Auth: auth,
			})
			if err != nil {
				fmt.Printf("failed to resolve repository %q\n", gitUrl)
				continue
			}

			acornfile := filepath.Join(workdir, vcs.Acornfile)
			if _, err := os.Stat(acornfile); err == nil {
				// Acornfile exists
			} else if errors.Is(err, os.ErrNotExist) {
				// Acornfile does not exist so we should create it
				err = os.WriteFile(acornfile, []byte(app.Status.Staged.AppImage.Acornfile), 0666)
				if err != nil {
					fmt.Printf("failed to create file %q in repository %q", acornfile, gitUrl)
					// TODO we hit an error state but already cloned the repo, should we clean up the repo we cloned?
					continue
				}
			} else {
				fmt.Printf("could not check for file %q in repository %q", acornfile, gitUrl)
				// TODO we hit an error state but already cloned the repo, should we clean up the repo we cloned?
				continue
			}

			imageSource = imagesource.NewImageSource(s.client.AcornConfigFile(), acornfile, s.ArgsFile, args, nil, z.Dereference(s.AutoUpgrade))
			break
		}
	}

	if !imageSource.IsImageSet() {
		return fmt.Errorf("failed to resolve image")
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
