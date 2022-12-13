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
	"github.com/google/go-containerregistry/pkg/name"
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
		image, err := c.ImageGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if strings.Contains(image.Digest, args[0]) && len(image.Tags) != 0 {
			args[0] = image.Tags[0]
		}

		images = []apiv1.Image{*image}
	} else {
		images, err = c.ImageList(cmd.Context())
		if err != nil {
			return err
		}
	}

	if a.Containers {
		return printContainerImages(images, cmd.Context(), c, a, args)
	}

	out := table.NewWriter(tables.ImageAcorn, system.UserNamespace(), false, a.Output)
	if a.Quiet {
		out = table.NewWriter([][]string{
			{"Name", "{{ .Name }}"},
		}, system.UserNamespace(), true, a.Output)
	}

	out.AddFormatFunc("trunc", func(str string) string {
		if a.NoTrunc {
			return str
		}
		return strings.TrimPrefix(str, "sha256:")[:12]
	})

	for _, image := range images {
		imagePrint := imagePrint{
			Name:   image.ObjectMeta.Name,
			Digest: image.Digest,
			Tag:    "",
		}
		for _, tag := range image.Tags {
			imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
			if err != nil {
				return err
			}
			if tag == "" && !a.All {
				continue
			}
			if tag == "" {
				out.Write(imagePrint)
				continue
			} else {
				imagePrint.Tag = imageParsedTag.TagStr()
			}

			imagePrint.Repository = imageParsedTag.RepositoryStr()
			out.Write(imagePrint)
		}
		if len(image.Tags) == 0 && a.All {
			out.Write(imagePrint)
		}
	}

	return out.Err()
}

type imagePrint struct {
	Name       string `json:"name,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tags,omitempty"`
}

type imageContainer struct {
	Container string
	Repo      string
	Tags      []string
	Digest    string
	ImageID   string
}

type imageContainerPrint struct {
	Container string
	Repo      string
	Tag       string
	Digest    string
	ImageID   string
}

func printContainerImages(images []apiv1.Image, ctx context.Context, c client.Client, a *Image, args []string) error {
	tagToMatch := ""
	if len(args) == 1 {
		tagToMatch = args[0]
	}
	out := table.NewWriter(tables.ImageContainer, system.UserNamespace(), a.Quiet, a.Output)

	if a.Quiet {
		out = table.NewWriter([][]string{
			{"Name", "{{if ne .Repo \"\"}}{{.Repo}}:{{end}}{{.Tag}}@{{.Digest}}"},
		}, system.UserNamespace(), a.Quiet, a.Output)
	}

	for _, image := range images {
		containerImages, err := getImageContainers(c, ctx, image)
		if err != nil {
			return err
		}
		for _, imageContainer := range containerImages {
			imageContainerPrint := imageContainerPrint{
				Container: imageContainer.Container,
				Repo:      imageContainer.Repo,
				Digest:    imageContainer.Digest,
				ImageID:   imageContainer.ImageID,
			}
			for _, tag := range imageContainer.Tags {
				imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
				if err != nil {
					continue
				}
				if tagToMatch == "" || tagToMatch == tag {
					if a.All || tag != "" {
						imageContainerPrint.Tag = imageParsedTag.TagStr()
						imageContainerPrint.Repo = imageParsedTag.RepositoryStr()
						out.Write(imageContainerPrint)
					}
				}
			}
			if len(imageContainer.Tags) == 0 && a.All {
				imageContainerPrint.Tag = "<none>"
				imageContainerPrint.Repo = "<none>"
				out.Write(imageContainerPrint)
			}
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
		imageContainerObject := imageContainer{
			Tags:      image.Tags,
			Container: k,
			Digest:    v.Image,
			ImageID:   image.Name,
		}

		imageContainers = append(imageContainers, imageContainerObject)

		for sidecar, img := range v.Sidecars {
			ic := imageContainer{
				Tags:      image.Tags,
				Container: sidecar,
				Digest:    img.Image,
				ImageID:   image.Name,
			}
			imageContainers = append(imageContainers, ic)
		}
	}

	return imageContainers
}
