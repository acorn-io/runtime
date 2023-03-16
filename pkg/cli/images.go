package cli

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

func NewImage(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Image{client: c.ClientFactory}, cobra.Command{
		Use:     "image [flags] [IMAGE_REPO:TAG|IMAGE_ID]",
		Aliases: []string{"images", "i"},
		Example: `
acorn images`,
		SilenceUsage:      true,
		Short:             "Manage images",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})
	cmd.AddCommand(NewImageDelete(c))
	return cmd
}

type Image struct {
	All        bool   `usage:"Include untagged images" short:"a" local:"true"`
	Quiet      bool   `usage:"Output only names" short:"q" local:"true"`
	NoTrunc    bool   `usage:"Don't truncate IDs" local:"true"`
	Output     string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o" local:"true"`
	Containers bool   `usage:"Show containers for images" short:"c" local:"true"`
	client     ClientFactory
}

func (a *Image) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	var images []apiv1.Image
	var image *apiv1.Image
	tagToMatch := ""

	allSetByUser := cmd.Flags().Changed("all")

	if len(args) == 1 {
		searchStr := args[0]

		ref, err := name.ParseReference(searchStr, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
		if err != nil {
			return err
		}

		// If the image is a digest, then we need to get the image by digest
		if dig, ok := ref.(name.Digest); ok {
			searchStr = dig.DigestStr()
		}

		image, err = c.ImageGet(cmd.Context(), searchStr)
		if err != nil {
			return err
		}
		if !strings.Contains(image.Digest, args[0]) {
			//normalize through ParseReference inorder to add :latest tag to input if necessary
			imageParsedReference, err := name.ParseReference(args[0], name.WithDefaultRegistry(""))
			if err != nil {
				return err
			}
			tagToMatch = imageParsedReference.Name()
		}
		images = []apiv1.Image{*image}

		// If an image was provided explicitly, then display it even if it doesn't have tags
		a.All = true
	} else {
		images, err = c.ImageList(cmd.Context())
		if err != nil {
			return err
		}
	}

	if a.Containers {
		//only display first tag in -c <tag>
		if image != nil && len(image.Tags) != 0 && tagToMatch == "" && len(args) != 0 {
			args[0] = image.Tags[0]
			tagToMatch = image.Tags[0]
		}

		return printContainerImages(images, cmd.Context(), c, a, args, tagToMatch)
	}

	out := table.NewWriter(tables.ImageAcorn, false, a.Output)
	if a.Quiet {
		out = table.NewWriter([][]string{
			{"Name", "{{ .Name }}"},
		}, true, a.Output)
	}

	out.AddFormatFunc("trunc", func(str string) string {
		if a.NoTrunc {
			return str
		}
		return strings.TrimPrefix(str, "sha256:")[:12]
	})

	for _, image := range images {
		imagePrint := imagePrint{
			Name:       image.ObjectMeta.Name,
			Digest:     image.Digest,
			Tag:        "",
			Repository: "",
		}
		if len(image.Tags) == 0 && a.All {
			out.Write(imagePrint)
			continue
		}
		for _, tag := range image.Tags {
			imageTagRef, err := name.ParseReference(tag, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
			if err != nil {
				return err
			}

			if imageTagRef.Identifier() == "" {
				tag = fmt.Sprintf("%s/%s@%s", imageTagRef.Context().RegistryStr(), imageTagRef.Context().RepositoryStr(), image.Digest)
				imageTagRef, err = name.ParseReference(tag, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
				if err != nil {
					return err
				}
			}

			if imageTagRef.Context().RegistryStr() != "" {
				imagePrint.Repository = imageTagRef.Context().RegistryStr() + "/"
			}
			imagePrint.Repository += imageTagRef.Context().RepositoryStr()

			if tagToMatch == imageTagRef.Name() {
				ntag, ok := imageTagRef.(name.Tag)
				if ok {
					imagePrint.Tag = ntag.TagStr()
				}
				out.Write(imagePrint)
			} else if tagToMatch == "" {
				ntag, ok := imageTagRef.(name.Tag)
				if ok {
					imagePrint.Tag = ntag.TagStr()
				} else {
					if !allSetByUser {
						continue
					}
				}
				out.Write(imagePrint)
			}
		}
	}

	return out.Err()
}

type imagePrint struct {
	Name       string `json:"name,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
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

func printContainerImages(images []apiv1.Image, ctx context.Context, c client.Client, a *Image, args []string, tagToMatch string) error {
	out := table.NewWriter(tables.ImageContainer, a.Quiet, a.Output)

	if a.Quiet {
		out = table.NewWriter([][]string{
			{"Name", "{{if ne .Repo \"\"}}{{.Repo}}:{{end}}{{.Tag}}@{{.Digest}}"},
		}, a.Quiet, a.Output)
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
			if len(imageContainer.Tags) == 0 && a.All {
				imageContainerPrint.Tag = "<none>"
				imageContainerPrint.Repo = "<none>"
				out.Write(imageContainerPrint)
				continue
			}
			for _, tag := range imageContainer.Tags {
				imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
				if err != nil {
					continue
				}
				if tagToMatch == imageParsedTag.Name() || tagToMatch == "" {
					imageContainerPrint.Tag = imageParsedTag.TagStr()
					if imageParsedTag.RegistryStr() != "" {
						imageContainerPrint.Repo = imageParsedTag.RegistryStr() + "/"
					}
					imageContainerPrint.Repo += imageParsedTag.RepositoryStr()
					out.Write(imageContainerPrint)
					imageContainerPrint.Repo = ""
				}
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
