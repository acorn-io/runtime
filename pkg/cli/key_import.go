package cli

import (
	"crypto"
	"fmt"
	"os"
	"strings"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	acornsign "github.com/acorn-io/acorn/pkg/cosign"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/spf13/cobra"
)

func NewKeyImport(c CommandContext) *cobra.Command {
	cmd := cli.Command(&KeyImport{client: c.ClientFactory}, cobra.Command{
		Use: "import [flags] KEYPATH",
		Example: `
acorn key import ~/.ssh/id_rsa`,
		SilenceUsage: true,
		Short:        "Import a (public) key",
		Args:         cobra.ExactArgs(1),
	})
	return cmd
}

type KeyImport struct {
	client ClientFactory
}

func (a *KeyImport) Run(cmd *cobra.Command, args []string) error {
	keyFile, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	keyData := strings.Fields(string(keyFile))[1]

	// Double verification that this is a valid and usable key
	key, err := acornsign.ParsePublicKey(keyData)
	if err != nil {
		return err
	}

	_, err = signature.LoadVerifier(key, crypto.SHA256) // TODO: make algorithm configurable
	if err != nil {
		return err
	}

	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	pk, err := c.KeyCreate(cmd.Context(), key)
	if err != nil {
		return err
	}

	if pk == nil {
		return fmt.Errorf("failed to import key")
	}

	fmt.Printf("Key %s imported\nFingerprint: %s\n", pk.Name, pk.Fingerprint)

	return nil
}
