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

func NewPull(c CommandContext) *cobra.Command {
	return cli.Command(&Pull{client: c.ClientFactory}, cobra.Command{
		Use:          "pull [flags] IMAGE",
		SilenceUsage: true,
		Short:        "Pull an image from a remote registry",
		Args:         cobra.ExactArgs(1),
	})
}

type Pull struct {
	client ClientFactory
}

func (s *Pull) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(args[0])
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

	auth, _, err := creds.Get(cmd.Context(), ref.Context().RegistryStr())
	if err != nil {
		return err
	}

	progress, err := c.ImagePull(cmd.Context(), args[0], &client.ImagePullOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}

	return progressbar.Print(progress)
}
