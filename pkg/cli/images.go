package cli

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
)

func NewImage(c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&Image{client: c.ClientFactory}, cobra.Command{
		Use:     "image [flags] [APP_NAME...]",
		Aliases: []string{"images", "i"},
		Example: `
acorn images`,
		SilenceUsage: true,
		Short:        "Manage images",
		Args:         cobra.MaximumNArgs(1),
	})
	cmd.AddCommand(NewImageDelete(c))
	return cmd
}

type Image struct {
	All        bool   `usage:"Include untagged images" short:"a"`
	Quiet      bool   `usage:"Output only names" short:"q"`
	NoTrunc    bool   `usage:"Don't truncate IDs"`
	Output     string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	Containers bool   `usage:"Show containers for images" short:"c"`
	client     client.ClientFactory
}

func (a *Image) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

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

	if a.Containers {
		return printContainerImages(images, cmd.Context(), c, a)
	}

	out := table.NewWriter(tables.Image, system.UserNamespace(), false, a.Output)
	if a.Quiet {
		out = table.NewWriter([][]string{
			{"Name", "{{ . | name }}"},
		}, system.UserNamespace(), true, a.Output)
	}

	out.AddFormatFunc("trunc", func(str string) string {
		if a.NoTrunc {
			return str
		}
		return strings.TrimPrefix(str, "sha256:")[:12]
	})

	for _, image := range images {
		if image.Tag == "" && image.Repository == "" && !a.All {
			continue
		}
		out.Write(image)
	}

	return out.Err()
}

type imageContainer struct {
	Container string
	Repo      string
	Tag       string
	Digest    string
	ImageID   string
}

func printContainerImages(images []apiv1.Image, ctx context.Context, c client.Client, a *Image) error {
	out := table.NewWriter(tables.ImageContainer, system.UserNamespace(), a.Quiet, a.Output)

	if a.Quiet {
		out = table.NewWriter([][]string{
			{"Name", "{{if ne .Repo \"\"}}{{.Repo}}:{{end}}{{.Tag}}@{{.Digest}}"},
		}, system.UserNamespace(), a.Quiet, a.Output)
	}

	for _, image := range images {
		if image.Tag == "" && image.Repository == "" {
			if !a.All {
				continue
			}
			if !a.Quiet {
				image.Tag = "<none>"
				image.Repository = "<none>"
			} else {
				image.Tag = image.Name
				image.Repository = "<none>"
			}
		}

		containerImages, err := getImageContainers(c, ctx, image)
		if err != nil {
			return err
		}

		for _, imgContainer := range containerImages {
			out.Write(imgContainer)
		}
	}

	return out.Err()
}

func getImageContainers(c client.Client, ctx context.Context, image apiv1.Image) ([]imageContainer, error) {
	imageContainers := []imageContainer{}

	imgDetails, err := c.ImageDetails(ctx, image.Name, nil)
	if err != nil {
		return imageContainers, err
	}

	imageData := imgDetails.AppImage.ImageData

	imageContainers = append(imageContainers, newImageContainerList(image, imageData.Containers)...)
	imageContainers = append(imageContainers, newImageContainerList(image, imageData.Jobs)...)

	return imageContainers, nil
}

func newImageContainerList(image apiv1.Image, containers map[string]v1.ContainerData) []imageContainer {
	imageContainers := []imageContainer{}

	for k, v := range containers {
		ImgContainer := imageContainer{
			Repo:      image.Repository,
			Tag:       image.Tag,
			Container: k,
			Digest:    v.Image,
			ImageID:   image.Name,
		}

		imageContainers = append(imageContainers, ImgContainer)

		for sidecar, img := range v.Sidecars {
			ic := imageContainer{
				Repo:      image.Repository,
				Tag:       image.Tag,
				Container: sidecar,
				Digest:    img.Image,
				ImageID:   image.Name,
			}
			imageContainers = append(imageContainers, ic)
		}
	}

	return imageContainers
}
