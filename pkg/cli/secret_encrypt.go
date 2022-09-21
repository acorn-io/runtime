package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewSecretEncrypt() *cobra.Command {
	cmd := cli.Command(&Encrypt{}, cobra.Command{
		Use:          "encrypt [flags] STRING",
		SilenceUsage: true,
		Short:        "Encrypt string information with clusters public key",
		Args:         cobra.RangeArgs(1, 1),
	})
	return cmd
}

type Encrypt struct {
	PublicKey []string `usage:"Pass one or more cluster publicKey values"`
}

func (e *Encrypt) Run(cmd *cobra.Command, args []string) error {
	out := table.NewWriter([][]string{
		{"Name", "{{.}}"},
	}, "", true, "")

	c, err := client.Default()
	if err != nil {
		return err
	}

	if len(args[0]) > 4096 {
		logrus.Fatal("Length of string data is too long to encrypt. Must be less than 4096 bytes.")
	}

	if len(e.PublicKey) == 0 {
		info, err := c.Info(cmd.Context())
		if err != nil {
			return err
		}
		e.PublicKey = append(e.PublicKey, info.Spec.PublicKey)
	}

	encData, err := nacl.MultipleKeyEncrypt(args[0], e.PublicKey)
	if err != nil {
		return err
	}

	output, err := encData.Marshal()
	if err != nil {
		return err
	}

	out.Write(output)

	return out.Err()
}
