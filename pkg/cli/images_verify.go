package cli

import (
	"errors"
	"fmt"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewImageVerify(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ImageVerify{client: c.ClientFactory}, cobra.Command{
		Use: "verify IMAGE_NAME [flags]",
		Example: `# Verify using a locally stored public key file
acorn image verify my-image --key ./my-key.pub

# Verify using a public key belonging to a GitHub Identity
acorn image verify my-image --key gh://ibuildthecloud

# Verify using a public key belonging to an Acorn Manager Identity
acorn image verify my-image --key ac://ibuildthecloud
`,
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
	Annotations map[string]string `usage:"Annotations to check for in the signature" short:"a" local:"true" name:"annotation"`
}

func (a *ImageVerify) Run(cmd *cobra.Command, args []string) error {
	if a.Key == "" {
		return fmt.Errorf("key is required")
	}

	targetName := args[0]
	targetDigest := ""

	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	img, tag, err := client.FindImage(cmd.Context(), c, targetName)
	if err != nil && !errors.As(err, &images.ErrImageNotFound{}) {
		return err
	}

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

	pterm.Info.Printf("Verifying Image %s (digest: %s) using key %s\n", targetName, targetDigest, a.Key)

	annotationRules := internalv1.SignatureAnnotations{
		Match: a.Annotations,
	}

	cc, err := c.GetClient()
	if err != nil {
		return err
	}

	verifyOpts := &acornsign.VerifyOpts{
		AnnotationRules:    annotationRules,
		SignatureAlgorithm: "sha256",
		Key:                a.Key,
		NoCache:            true,
	}
	if err := verifyOpts.WithRemoteOpts(cmd.Context(), cc, c.GetNamespace()); err != nil {
		pterm.Debug.Printf("Error getting remote opts: %v\n", err)
		pterm.Warning.Println("Unable to get remote opts for registry authentication, trying without.")
	}

	if err := acornsign.EnsureReferences(cmd.Context(), cc, target.String(), verifyOpts); err != nil {
		return err
	}

	if err := acornsign.VerifySignature(cmd.Context(), *verifyOpts); err != nil {
		return err
	}

	pterm.Success.Println("Signature verified")

	return nil
}
