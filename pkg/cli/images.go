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
	"github.com/acorn-io/acorn/pkg/progressbar"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	All        bool   `usage:"Include untagged images" short:"a"`
	Quiet      bool   `usage:"Output only names" short:"q"`
	NoTrunc    bool   `usage:"Don't truncate IDs"`
	Output     string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	Containers bool   `usage:"Show containers for images" short:"c"`
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

	if a.Containers {
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
	Digest    string
}

func printContainerImages(images []apiv1.Image, ctx context.Context, c client.Client, output string, quiet bool) error {
	out := table.NewWriter(tables.ImageContainer, system.UserNamespace(), quiet, output)

	if quiet {
		out = table.NewWriter([][]string{
			{"Name", "{{.Repo}}:{{.Tag}}@{{.Digest}}"},
		}, system.UserNamespace(), quiet, output)
	}

	for _, image := range images {
		if image.Tag == "" && image.Repository == "" {
			continue
		}

		containerImages, err := getImageContainers(c, ctx, image)
		if err != nil {
			logrus.Error(err)
			//continue
		}
		for _, imgContainer := range containerImages {
			out.Write(imgContainer)
		}

	}

	return out.Err()
}

func getImageContainers(c client.Client, ctx context.Context, image apiv1.Image) ([]ImageContainer, error) {
	imageContainers := []ImageContainer{}

	imgDetails, err := c.ImageDetails(ctx, image.Name, nil)
	if err != nil {
		return imageContainers, err
	}

	imageData := imgDetails.AppImage.ImageData

	imageContainers = append(imageContainers, newImageContainerList(image, imageData.Containers)...)
	imageContainers = append(imageContainers, newImageContainerList(image, imageData.Jobs)...)

	for _, acorn := range imageData.Acorns {
		acornImg := apiv1.Image{
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.TrimPrefix(acorn.Image, "sha256:"),
			},
			Repository: image.Repository,
			Tag:        image.Tag,
		}

		acornImageContainers, err := getImageContainers(c, ctx, acornImg)
		if apierrors.IsNotFound(err) {
			pullImage := fmt.Sprintf("%s:%s@sha256:%s", acornImg.Repository, acornImg.Tag, acornImg.Name)
			progress, err := c.ImagePull(ctx, pullImage, nil)
			if err != nil {
				return imageContainers, err
			}
			err = progressbar.Print(progress)
			if err != nil {
				return imageContainers, err
			}
		} else if err != nil {
			return imageContainers, err
		}
		imageContainers = append(imageContainers, acornImageContainers...)
	}

	return imageContainers, nil
}

func newImageContainerList(image apiv1.Image, containers map[string]v1.ContainerData) []ImageContainer {
	imageContainers := []ImageContainer{}

	for k, v := range containers {
		ImgContainer := ImageContainer{
			Repo:      image.Repository,
			Tag:       image.Tag,
			Container: k,
			Digest:    v.Image,
		}

		imageContainers = append(imageContainers, ImgContainer)

		for sidecar, img := range v.Sidecars {
			ic := ImageContainer{
				Repo:      image.Repository,
				Tag:       image.Tag,
				Container: sidecar,
				Digest:    img.Image,
			}
			imageContainers = append(imageContainers, ic)
		}
	}

	return imageContainers
}
