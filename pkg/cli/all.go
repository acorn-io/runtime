package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/spf13/cobra"
)

func NewAll(c CommandContext) *cobra.Command {
	return cli.Command(&All{client: c.ClientFactory}, cobra.Command{
		Use: "all",
		Example: `
acorn all`,
		SilenceUsage: true,
		Short:        "List (almost) all objects",
	})
}

type All struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	Images bool   `usage:"Include images in output" short:"i"`
	All    bool   `usage:"Include stopped apps/containers" short:"a"`
	client ClientFactory
}

func (a *All) Run(cmd *cobra.Command, args []string) error {
	if !a.Quiet {
		fmt.Println("")
		fmt.Println("ACORNS:")
	}
	ps := &Ps{
		Quiet:  a.Quiet,
		Output: a.Output,
		All:    a.All,
		client: a.client,
	}
	psErr := ps.Run(cmd, nil)

	con := &Container{
		Quiet:  a.Quiet,
		Output: a.Output,
		All:    a.All,
		client: a.client,
	}
	if !a.Quiet {
		fmt.Println("")
		fmt.Println("CONTAINERS:")
	}
	conErr := con.Run(cmd, nil)

	vol := &Volume{
		Quiet:  a.Quiet,
		Output: a.Output,
		client: a.client,
	}
	if !a.Quiet {
		fmt.Println("")
		fmt.Println("VOLUMES:")
	}
	volErr := vol.Run(cmd, nil)

	sec := &Secret{
		Quiet:  a.Quiet,
		Output: a.Output,
		client: a.client,
	}
	if !a.Quiet {
		fmt.Println("")
		fmt.Println("SECRETS:")
	}
	secErr := sec.Run(cmd, nil)

	var imgErr error

	if a.Images {
		img := &Image{
			Quiet:  a.Quiet,
			Output: a.Output,
			All:    a.All,
			client: a.client,
		}
		if !a.Quiet {
			fmt.Println("")
			fmt.Println("IMAGES:")
		}
		imgErr = img.Run(cmd, nil)
	}

	return merr.NewErrors(psErr, conErr, volErr, secErr, imgErr)
}
