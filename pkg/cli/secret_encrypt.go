package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewSecretEncrypt(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Encrypt{client: c.ClientFactory}, cobra.Command{
		Use:          "encrypt [flags] STRING",
		SilenceUsage: true,
		Short:        "Encrypt string information with clusters public key",
		Args:         cobra.MaximumNArgs(1),
	})
	return cmd
}

type Encrypt struct {
	PlaintextStdin bool     `usage:"Take the plaintext from stdin"`
	PublicKey      []string `usage:"Pass one or more cluster publicKey values"`
	client         ClientFactory
}

func (e *Encrypt) Run(cmd *cobra.Command, args []string) error {
	out := table.NewWriter([][]string{
		{"Name", "{{.}}"},
	}, true, "")
	c, err := e.client.CreateDefault()
	if err != nil {
		return err
	}

	if e.PlaintextStdin && len(args) == 0 {
		contents, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		plaintext := strings.TrimSuffix(string(contents), "\n")
		plaintext = strings.TrimSuffix(plaintext, "\r")

		args = append(args, plaintext)
	} else if e.PlaintextStdin && len(args) > 0 {
		return fmt.Errorf("no args can be provided if using stdin")
	}

	if len(args) == 1 && len(args[0]) > 4096 {
		logrus.Fatal("Length of string data is too long to encrypt. Must be less than 4096 bytes.")
	}

	var q []*survey.Question
	if len(args) == 0 {
		q = append(q, &survey.Question{
			Name:     "plaintext",
			Prompt:   &survey.Password{Message: "Data to encrypt"},
			Validate: survey.MaxLength(4096),
		})
	}

	plaintext := ""
	if err := survey.Ask(q, &plaintext); err != nil {
		return err
	}

	args = append(args, plaintext)

	if len(e.PublicKey) == 0 {
		fullInfo, err := c.Info(cmd.Context())
		if err != nil {
			return err
		}
		for _, info := range fullInfo {
			for _, key := range info.Spec.PublicKeys {
				e.PublicKey = append(e.PublicKey, key.KeyID)
			}
		}
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
