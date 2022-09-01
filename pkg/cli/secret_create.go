package cli

import (
	"fmt"
	"os"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/spf13/cobra"
)

func NewSecretCreate() *cobra.Command {
	cmd := cli.Command(&SecretCreate{}, cobra.Command{
		Use: "create [flags] SECRET_NAME",
		Example: `
# Create secret with specific keys
acorn secret create --data key-name=value --data key-name2=value2 my-secret

# Read full secret from a file
acorn secret create --file secret.yaml my-secret

# Read key value from a file
acorn secret create --data @key-name=secret.yaml my-secret`,
		SilenceUsage: true,
		Short:        "Create a secret",
		Args:         cobra.ExactArgs(1),
	})
	return cmd
}

type SecretCreate struct {
	Data []string `usage:"Secret data format key=value or @key=filename to read from file"`
	File string   `usage:"File to read for entire secret in cue/yaml/json format"`
	Type string   `usage:"Secret type"`
}

func (a *SecretCreate) buildSecret() (*apiv1.Secret, error) {
	secret := &struct {
		apiv1.Secret `json:",inline"`
		StringData   map[string]string `json:"stringData,omitempty"`
	}{}

	if a.File != "" {
		err := cue.UnmarshalFile(a.File, secret)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", a.File, err)
		}
		for k, v := range secret.StringData {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}
			secret.Data[k] = []byte(v)
		}
	}

	for _, kv := range a.Data {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid data format [%s] must be in key=value form", kv)
		}
		if strings.HasPrefix(key, "@") {
			key = key[1:]
			content, err := os.ReadFile(value)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", value, err)
			}
			value = string(content)
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[key] = []byte(value)
	}

	if a.Type != "" {
		secret.Type = a.Type
	}

	return &secret.Secret, nil
}

func (a *SecretCreate) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	secret, err := a.buildSecret()
	if err != nil {
		return err
	}

	newSecret, err := client.SecretCreate(cmd.Context(), args[0], secret.Type, secret.Data)
	if err != nil {
		return err
	}

	fmt.Println(newSecret.Name)
	return nil
}
