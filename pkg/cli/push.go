package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/credentials"
	"github.com/acorn-io/acorn/pkg/progressbar"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

func NewPush(c CommandContext) *cobra.Command {
	return cli.Command(&Push{client: c.ClientFactory}, cobra.Command{
		Use:               "push [flags] IMAGE",
		SilenceUsage:      true,
		Short:             "Push an image to a remote registry",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(false)).withShouldCompleteOptions(onlyNumArgs(1)).checkProjectPrefix().complete,
	})
}

type Push struct {
	client ClientFactory
}

func (s *Push) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	cfg, err := config.ReadCLIConfig()
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

	prog, err := c.ImagePush(cmd.Context(), args[0], &client.ImagePushOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}

	return progressbar.Print(prog)
}
