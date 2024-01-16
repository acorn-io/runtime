package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/acorn-io/aml"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func NewSecretCreate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&SecretCreate{client: c.ClientFactory}, cobra.Command{
		Use: "create [flags] SECRET_NAME",
		Example: `
# Create secret with specific keys
acorn secret create --data key-name=value --data key-name2=value2 my-secret

# Read full secret from a file. The file should have a type and data field.
acorn secret create --file secret.yaml my-secret

# Read key value from a file
acorn secret create --data @key-name=secret.yaml my-secret`,
		SilenceUsage: true,
		Short:        "Create a secret",
		Args:         cobra.ExactArgs(1),
	})
	return cmd
}

type SecretFactory struct {
	Data []string `usage:"Secret data format key=value or @key=filename to read from file"`
	File string   `usage:"File to read for entire secret in aml/yaml/json format"`
	Type string   `usage:"Secret type"`
}

type SecretCreate struct {
	SecretFactory
	Update  bool `usage:"Update the secret if it already exists" short:"u"`
	Replace bool `usage:"Replace the secret with only defined values, resetting undefined fields to default values" json:"replace,omitempty"`
	client  ClientFactory
}

func (a *SecretFactory) buildSecret() (*apiv1.Secret, error) {
	secret := &struct {
		apiv1.Secret `json:",inline"`
		StringData   map[string]string `json:"stringData,omitempty"`
	}{}

	if a.File != "" {
		err := aml.UnmarshalFile(a.File, secret)
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
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	secret, err := a.buildSecret()
	if err != nil {
		return err
	}

	newSecret, err := client.SecretCreate(cmd.Context(), args[0], secret.Type, secret.Data)
	if apierrors.IsAlreadyExists(err) {
		if a.Replace {
			newSecret, err = client.SecretUpdate(cmd.Context(), args[0], secret.Data)
			if err != nil {
				return err
			}
		} else if a.Update {
			existing, err := client.SecretReveal(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if existing.Data == nil {
				existing.Data = map[string][]byte{}
			}
			for k, v := range secret.Data {
				existing.Data[k] = v
			}
			newSecret, err = client.SecretUpdate(cmd.Context(), args[0], existing.Data)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("secret %s already exists", args[0])
		}
	} else if err != nil {
		return err
	}

	fmt.Println(newSecret.Name)
	return nil
}
