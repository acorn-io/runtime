package cli

import (
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler-cli/pkg/table"
	"github.com/sirupsen/logrus"
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
	Quiet   bool   `desc:"Output only names" short:"q"`
	NoTrunc bool   `desc:"Don't truncate IDs"`
	Output  string `desc:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

type ImageDisplay struct {
	Repository string
	Tag        string
	ImageID    string
}

func (a *Image) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	out := table.NewWriter([][]string{
		{"REPOSITORY", "Repository"},
		{"TAG", "Tag"},
		{"IMAGEID", "{{trunc .ImageID}}"},
	}, "", false, a.Output)
	if a.Quiet {
		out = table.NewWriter([][]string{
			{"NAME", "ImageID"},
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
		if len(image.Tags) == 0 {
			out.Write(ImageDisplay{
				Repository: "<none>",
				Tag:        "<none>",
				ImageID:    image.Digest,
			})
		} else {
			for _, tag := range image.Tags {
				ref, err := name.NewTag(tag)
				if err != nil {
					logrus.Errorf("invalid tag [%s]: %v", tag, err)
					continue
				}
				out.Write(ImageDisplay{
					Repository: strings.TrimPrefix(ref.Context().RepositoryStr(), "library/"),
					Tag:        ref.TagStr(),
					ImageID:    image.Digest,
				})
			}
		}
	}

	return out.Err()
}
