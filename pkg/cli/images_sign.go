package cli

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pterm/pterm"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/signature"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
	"github.com/spf13/cobra"

	"github.com/acorn-io/runtime/pkg/prompt"
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
	client      ClientFactory
	Key         string            `usage:"Key to use for signing" short:"k" local:"true"`
	Annotations map[string]string `usage:"Annotations to add to the signature" short:"a" local:"true" name:"annotation"`
}

func (a *ImageSign) Run(cmd *cobra.Command, args []string) error {
	if a.Key == "" {
		return fmt.Errorf("key is required")
	}

	imageName := args[0]

	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	auth, err := getAuthForImage(cmd.Context(), a.client, imageName)
	if err != nil {
		return err
	}

	// not failing here, since it could be a local image
	ref, _ := name.ParseReference(imageName)

	details, err := c.ImageDetails(cmd.Context(), args[0], &client.ImageDetailsOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}

	targetDigest := ref.Context().Digest(details.AppImage.Digest)

	pterm.Info.Printf("Signing Image %s (digest: %s) using key %s\n", imageName, targetDigest, a.Key)

	pass, err := getPrivateKeyPass(a.Key)
	if err != nil {
		return err
	}
	if len(pass) == 0 {
		pass = nil // nothing instead of empty pass
	}

	pf := func(_ bool) ([]byte, error) {
		return pass, nil
	}

	// Get a sigSigner-verifier from a private key and if the key type is not supported, try to import it first
	var sigSigner sigsig.SignerVerifier
	sigSigner, err = signature.SignerVerifierFromKeyRef(cmd.Context(), a.Key, pf)
	if err != nil {
		if !strings.Contains(err.Error(), "unsupported pem type") {
			return err
		}
		pterm.Debug.Printf("Key %s is not a supported PEM key, importing...\n", a.Key)
		keyBytes, err := acornsign.ImportKeyPair(a.Key, pass)
		if err != nil {
			return err
		}
		sigSigner, err = cosign.LoadPrivateKey(keyBytes.PrivateBytes, keyBytes.Password())
		if err != nil {
			return err
		}
	}

	var annotations map[string]interface{}
	if a.Annotations != nil {
		annotations = make(map[string]interface{}, len(a.Annotations))
		for k, v := range a.Annotations {
			annotations[k] = v
		}
	}

	payload, signature, err := sigsig.SignImage(sigSigner, targetDigest, annotations)
	if err != nil {
		return err
	}

	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	imageSignOpts := &client.ImageSignOptions{
		Auth: auth,
	}

	pubkey, err := sigSigner.PublicKey()
	if err != nil {
		return err
	}

	if pubkey != nil {
		pem, _, err := acornsign.PemEncodeCryptoPublicKey(pubkey)
		if err != nil {
			return err
		}

		imageSignOpts.PublicKey = string(pem)
	}

	sig, err := c.ImageSign(cmd.Context(), imageName, payload, signatureB64, imageSignOpts)
	if err != nil {
		return err
	}

	pterm.Success.Printf("Created signature %s\n", sig.SignatureDigest)

	return nil
}

// Get password for private key from environment, prompt or stdin (piped)
// Adapted from Cosign's readPasswordFn
func getPrivateKeyPass(keyfile string) ([]byte, error) {
	pw, ok := os.LookupEnv("ACORN_IMAGE_SIGN_PASSWORD")
	switch {
	case ok:
		return []byte(pw), nil
	case isTerm():
		return prompt.Password(fmt.Sprintf("Enter password for private key %s:", keyfile))
	default:
		return io.ReadAll(os.Stdin)
	}
}

func isTerm() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}
