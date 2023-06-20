package cli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	acornsign "github.com/acorn-io/acorn/pkg/cosign"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/generate"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/cosign/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/cosign/v2/pkg/signature"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
	"github.com/spf13/cobra"
)

func NewImageSign(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ImageSign{client: c.ClientFactory}, cobra.Command{
		Use:               "sign IMAGE_NAME [flags]",
		Example:           `acorn image sign my-image --key ./my-key`,
		SilenceUsage:      true,
		Short:             "Sign an Image",
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).complete,
		Args:              cobra.ExactArgs(1),
	})
	_ = cmd.MarkFlagFilename("key")
	return cmd
}

type ImageSign struct {
	client              ClientFactory
	Key                 string            `usage:"Key to use for signing" short:"k" local:"true" default:"./cosign.key"`
	Annotations         map[string]string `usage:"Annotations to add to the signature" short:"a" local:"true"`
	SignatureRepository string            `usage:"Repository to push the signature to" short:"r" local:"true" default:""`
	Push                bool              `usage:"Push the signature to the signature repository" short:"p" local:"true" default:"true"`
}

func (a *ImageSign) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	push := cmd.Flag("push").Changed && a.Push

	img, tag, err := client.FindImage(cmd.Context(), c, args[0])
	if err != nil && !errors.As(err, &images.ErrImageNotFound{}) {
		return err
	}

	targetName := args[0]
	targetDigest := ""

	if err == nil && tag == "" {
		return fmt.Errorf("Signing a local image without specifying the repository is not supported")
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
	fmt.Printf("Signing Image %s (digest: %s) using key %s\n", targetName, targetDigest, a.Key)

	pass, err := generate.GetPass(false)
	if err != nil {
		return err
	}

	pf := func(_ bool) ([]byte, error) {
		return pass, nil
	}

	// Get a sigSigner-verifier from a private key and if the key type is not supported, try to import it first
	var sigSigner sigsig.SignerVerifier
	sigSigner, err = signature.SignerVerifierFromKeyRef(cmd.Context(), a.Key, pf) // TODO(iwilltry42): use our own style password prompt
	if err != nil {
		if !strings.Contains(err.Error(), "unsupported pem type") {
			return err
		}
		fmt.Printf("Key %s is not a supported PEM key, importing...\n", a.Key)
		keyBytes, err := cosign.ImportKeyPair(a.Key, pf)
		if err != nil {
			return err
		}
		sigSigner, err = cosign.LoadPrivateKey(keyBytes.PrivateBytes, keyBytes.Password())
		if err != nil {
			return err
		}
	}

	dupeDetector := remote.NewDupeDetector(sigSigner)

	var annotations map[string]interface{}
	if a.Annotations != nil {
		annotations = make(map[string]interface{})
		for k, v := range a.Annotations {
			annotations[k] = v
		}
	}

	payload, signature, err := sigsig.SignImage(sigSigner, target, annotations)
	if err != nil {
		return err
	}

	signatureB64 := base64.StdEncoding.EncodeToString(signature)
	signatureOCI, err := static.NewSignature(payload, signatureB64)
	if err != nil {
		return err
	}

	ociEntity, err := ociremote.SignedEntity(ref) // TODO: here and in other places we may want to add remote opts, especially for registry auth
	if err != nil {
		return fmt.Errorf("accessing entity: %w", err)
	}

	mutatedOCIEntity, err := mutate.AttachSignatureToEntity(ociEntity, signatureOCI, mutate.WithDupeDetector(dupeDetector))
	if err != nil {
		return err
	}

	if push {
		targetRepo := ref.Context()
		if a.SignatureRepository != "" {
			ref, err := name.ParseReference(a.SignatureRepository)
			if err != nil {
				return err
			}
			targetRepo = ref.Context()
		}
		fmt.Printf("Pushing signature to %s\n", targetRepo.String())
		return ociremote.WriteSignatures(targetRepo, mutatedOCIEntity) // TODO: need remote opts
	}

	fmt.Println("Not pushing signature")

	return nil
}
