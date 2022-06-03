package cli

import (
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
)

func NewImage() *cobra.Command {
	return cli.Command(&Image{}, cobra.Command{
		Use:     "image [flags] [APP_NAME...]",
		Aliases: []string{"images", "i"},
		Example: `
acorn images`,
		SilenceUsage: true,
		Short:        "List images",
		Args:         cobra.MaximumNArgs(1),
	})
}

type Image struct {
	Quiet   bool   `usage:"Output only names" short:"q"`
	NoTrunc bool   `usage:"Don't truncate IDs"`
	Output  string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

func (a *Image) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Image, "", false, a.Output)
	if a.Quiet {
		out = table.NewWriter([][]string{
			{"NAME", "Name"},
		}, "", true, a.Output)
	}

	out.AddFormatFunc("trunc", func(str string) string {
		if a.NoTrunc {
			return str
		}
		return strings.TrimPrefix(str, "sha256:")[:12]
	})

	var images []apiv1.Image
	if len(args) == 1 {
		img, err := c.ImageGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		images = []apiv1.Image{*img}
	} else {
		images, err = c.ImageList(cmd.Context())
		if err != nil {
			return err
		}
	}

	for _, image := range images {
		out.Write(image)
	}

	return out.Err()
}
