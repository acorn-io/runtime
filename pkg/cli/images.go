package cli

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	cmd.AddCommand(NewImageDetail(c))
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
	repoToMatch := ""

	if len(args) == 1 {
		ref, err := name.ParseReference(args[0], name.WithDefaultRegistry(""), name.WithDefaultTag(""))
		if err != nil {
			return err
		}

		if ref.Identifier() != "" {
			// > search by ID, tag or prefix
			image, err = c.ImageGet(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if _, ok := ref.(name.Digest); ok {
				// > search by digest -> show all images
				repoToMatch = ref.Context().Name()
			} else if !strings.Contains(image.Digest, args[0]) {
				tagToMatch = ref.Name()
			}
			images = []apiv1.Image{*image}
		} else {
			if tags.SHAPermissivePrefixPattern.MatchString(args[0]) {
				// > search by ID or prefix or untagged (in order untagged > ID > prefix)
				image, err = c.ImageGet(cmd.Context(), args[0])
				if err != nil {
					if !apierrors.IsNotFound(err) {
						return err
					}
				} else if image != nil {
					images = []apiv1.Image{*image}

					// here, we don't know whether the image was matched by ID or by `:latest` auto-tag,
					// so we check for the latter in the tags, as it takes precedence
					tagRef, err := name.ParseReference(args[0], name.WithDefaultRegistry(""))
					if err != nil {
						return err
					}
					for _, t := range image.Tags {
						if t == tagRef.Name() {
							tagToMatch = t
							break
						}
					}
				}
			} else {
				// > search by repository
				// no tag or digest was provided, so we need to get all images and filter by repo
				repoToMatch = ref.Context().Name()
				il, err := c.ImageList(cmd.Context())
				if err != nil {
					return err
				}

				// only consider images that have at least one tag matching the registry/repo prefix -> granular filtering is done later
				for _, i := range il {
					for _, t := range i.Tags {
						if strings.HasPrefix(t, repoToMatch) {
							images = append(images, i)
							break
						}
					}
				}
			}
		}

		// If an image was provided explicitly, then display it even if it doesn't have tags
		a.All = true
	} else {
		// > No filter, list all
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

		// no tag set at all, so only print if --all is set
		if len(image.Tags) == 0 && a.All {
			out.Write(imagePrint)
			continue
		}

		// loop through all tags
		for _, tag := range image.Tags {
			imageTagRef, err := name.ParseReference(tag, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
			if err != nil {
				return err
			}

			// if we are searching by repo, then filter out tags that don't match
			if repoToMatch != "" && imageTagRef.Context().Name() != repoToMatch {
				continue
			}

			// in any case, add registry/repository information to the output
			if imageTagRef.Context().RegistryStr() != "" {
				imagePrint.Repository = imageTagRef.Context().RegistryStr() + "/"
			}
			imagePrint.Repository += imageTagRef.Context().RepositoryStr()

			if tagToMatch == "" && repoToMatch == "" {
				// > not searching by tag or repo
				if ntag, ok := imageTagRef.(name.Tag); ok {
					// it's a tag, so add the tag output
					imagePrint.Tag = ntag.TagStr()
				}
				out.Write(imagePrint)
			} else if tagToMatch == imageTagRef.Name() || repoToMatch == imageTagRef.Context().Name() {
				// > searching by tag
				if ntag, ok := imageTagRef.(name.Tag); ok {
					// it's a tag, so add the tag output
					imagePrint.Tag = ntag.TagStr()
					out.Write(imagePrint)
				} else if _, ok := imageTagRef.(name.Digest); ok {
					// it's a digest, so print without a tag
					out.Write(imagePrint)
				}
			}

			imagePrint.Repository = ""
			imagePrint.Tag = ""
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
