package cli

import (
	"errors"
	"fmt"
	"strings"

	internalv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	acornsign "github.com/acorn-io/acorn/pkg/cosign"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewImageVerify(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ImageVerify{client: c.ClientFactory}, cobra.Command{
		Use:               "verify IMAGE_NAME [flags]",
		Example:           `acorn image verify my-image --key ./my-key.pub`,
		SilenceUsage:      true,
		Short:             "Verify Image Signatures",
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).complete,
		Args:              cobra.ExactArgs(1),
	})
	_ = cmd.MarkFlagFilename("key")
	return cmd
}

type ImageVerify struct {
	client      ClientFactory
	Key         string            `usage:"Key to use for verifying" short:"k" local:"true" default:"./cosign.pub"`
	Annotations map[string]string `usage:"Annotations to check for in the signature" short:"a" local:"true"`
}

func (a *ImageVerify) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	img, tag, err := client.FindImage(cmd.Context(), c, args[0])
	if err != nil && !errors.As(err, &images.ErrImageNotFound{}) {
		return err
	}

	targetName := args[0]
	targetDigest := ""

	if err == nil && tag == "" {
		return fmt.Errorf("Verifying a local image without specifying the repository is not supported")
	} else if tag != "" {
		targetName = tag
		targetDigest = img.Digest
	}

	ref, err := name.ParseReference(targetName)
	if err != nil {
		return err
	}

	if targetDigest == "" {
		targetDigest, err = acornsign.SimpleDigest(ref)
		if err != nil {
			return err
		}
	}

	target := ref.Context().Digest(targetDigest)

	if a.Key == "" { // TODO: use default identity if no key is given
		return fmt.Errorf("key is required")
	}
	pterm.Info.Printf("Verifying Image %s (digest: %s) using key %s\n", targetName, targetDigest, a.Key)

	annotationRules := internalv1.SignatureAnnotations{
		Match: a.Annotations,
	}

	verifyOpts := acornsign.VerifyOpts{ // TODO: add ociremote opts
		AnnotationRules:    annotationRules,
		SignatureAlgorithm: "sha256",
		Key:                a.Key,
		NoCache:            true,
	}

	if strings.HasPrefix(a.Key, "ac://") {
		key, err := c.KeyGet(cmd.Context(), strings.TrimPrefix(a.Key, "ac://"))
		if err != nil {
			return err
		}
		verifyOpts.Key = key.Key
	}

	cc, err := c.GetClient()
	if err != nil {
		return err
	}
	if err := acornsign.EnsureReferences(cmd.Context(), cc, target.String(), &verifyOpts); err != nil {
		return err
	}

	if err := acornsign.VerifySignature(cmd.Context(), verifyOpts); err != nil {
		return err
	}

	pterm.Success.Println("Signature verified")

	return nil
}
