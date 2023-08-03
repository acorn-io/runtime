package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/progressbar"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewPush(c CommandContext) *cobra.Command {
	return cli.Command(&Push{client: c.ClientFactory}, cobra.Command{
		Use:               "push [flags] IMAGE",
		SilenceUsage:      true,
		Short:             "Push an image to a remote registry",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(false)).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})
}

type Push struct {
	client               ClientFactory
	Sign                 bool              `usage:"Sign the image before pushing" short:"s" local:"true" default:"false"`
	Key                  string            `usage:"Key to use for signing" short:"k" local:"true" default:"./cosign.key"`
	SignatureAnnotations map[string]string `usage:"Annotations to add to the signature" short:"a" local:"true" name:"signature-annotation"`
}

func (s *Push) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	cfg, err := s.client.Options().CLIConfig()
	if err != nil {
		return err
	}

	creds, err := credentials.NewStore(cfg, c)
	if err != nil {
		return err
	}

	tag, err := name.NewTag(args[0])
	if err != nil {
		return err
	}

	auth, _, err := creds.Get(cmd.Context(), tag.RegistryStr())
	if err != nil {
		return err
	}

	if s.Sign {
		sign := ImageSign{
			client:      s.client,
			Key:         s.Key,
			Annotations: s.SignatureAnnotations,
		}
		if err := sign.Run(cmd, args); err != nil {
			return err
		}

		pterm.Success.Printf("Signed %s\n", args[0])
	}

	prog, err := c.ImagePush(cmd.Context(), args[0], &client.ImagePushOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}

	if err := progressbar.Print(prog); err != nil {
		return err
	}

	return nil
}
