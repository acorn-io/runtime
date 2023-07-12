package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/progressbar"
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
	client      ClientFactory
	Verify      bool              `usage:"Verify the image signature BEFORE pulling and only pull on success" short:"v" local:"true" default:"false"`
	Key         string            `usage:"Key to use for verifying" short:"k" local:"true" default:"./cosign.pub"`
	Annotations map[string]string `usage:"Annotations to check for during verification" short:"a" local:"true" name:"annotation"`
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

	cfg, err := s.client.Options().CLIConfig()
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

	if s.Verify {
		v := ImageVerify{
			client:      s.client,
			Key:         s.Key,
			Annotations: s.Annotations,
		}
		if err := v.Run(cmd, args); err != nil {
			return fmt.Errorf("NOT pulling image: %w", err)
		}
	}

	progress, err := c.ImagePull(cmd.Context(), args[0], &client.ImagePullOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}

	return progressbar.Print(progress)
}
