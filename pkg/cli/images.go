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
	All       bool   `usage:"Include untagged images" short:"a"`
	Quiet     bool   `usage:"Output only names" short:"q"`
	NoTrunc   bool   `usage:"Don't truncate IDs"`
	Output    string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	Container bool   `usage:"Show containers for images" short:"c"`
}

func (a *Image) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
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

	if a.Container {
		return printContainerImages(images, cmd.Context(), c, a.Output, a.Quiet)
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

type ImageContainer struct {
	Container string
	Repo      string
	Tag       string
	Image     string
}

func printContainerImages(images []apiv1.Image, ctx context.Context, c client.Client, output string, quiet bool) error {
	out := table.NewWriter(tables.ImageContainers, system.UserNamespace(), quiet, output)

	if quiet {
		out = table.NewWriter([][]string{
			{"Name", "{{.Repo}}:{{.Tag}}@{{.Image}}"},
		}, system.UserNamespace(), quiet, output)
	}

	for _, image := range images {
		imgDetails, err := c.ImageDetails(ctx, image.Name, &client.ImageDetailsOptions{})
		if err != nil {
			return err
		}

		if image.Tag == "" && image.Repository == "" {
			continue
		}

		for _, imgContainer := range newImageContainer(image, imgDetails.AppImage.ImageData) {
			out.Write(imgContainer)
		}

	}

	return out.Err()
}

func newImageContainer(image apiv1.Image, imageData v1.ImagesData) []ImageContainer {
	imageContainers := []ImageContainer{}

	imageContainers = append(imageContainers, parseContainerData(image.Repository, image.Tag, imageData.Containers)...)
	imageContainers = append(imageContainers, parseContainerData(image.Repository, image.Tag, imageData.Jobs)...)

	return imageContainers
}

func parseContainerData(repo, tag string, containers map[string]v1.ContainerData) []ImageContainer {
	imageContainers := []ImageContainer{}

	for k, v := range containers {
		ImgContainer := ImageContainer{
			Repo:      repo,
			Tag:       tag,
			Container: k,
			Image:     v.Image,
		}

		imageContainers = append(imageContainers, ImgContainer)

		for sidecar, img := range v.Sidecars {
			ic := ImageContainer{
				Repo:      repo,
				Tag:       tag,
				Container: sidecar,
				Image:     img.Image,
			}
			imageContainers = append(imageContainers, ic)
		}
	}

	return imageContainers
}
