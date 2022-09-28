package cli

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/acorn/pkg/version"
	bversion "github.com/acorn-io/baaah/pkg/version"
	"github.com/spf13/cobra"
)

func NewInfo() *cobra.Command {
	cmd := cli.Command(&Info{}, cobra.Command{
		Use:          "info",
		SilenceUsage: true,
		Short:        "Info about acorn installation",
		Args:         cobra.NoArgs,
	})
	return cmd
}

type Info struct {
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o" default:"yaml"`
}

type ClientServerVersion struct {
	Client struct {
		Version bversion.Version `json:"version,omitempty"`
	} `json:"client,omitempty"`
	Server    apiv1.InfoSpec `json:"server,omitempty"`
	Namespace struct {
		PublicKeys []apiv1.EncryptionKeySpec `json:"publicKeys,omitempty"`
	} `json:"namespace,omitempty"`
}

func (s *Info) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	info, err := c.Info(cmd.Context())
	if err != nil {
		return err
	}

	//Formatting...
	ns := struct {
		PublicKeys []apiv1.EncryptionKeySpec `json:"publicKeys,omitempty"`
	}{PublicKeys: info.Spec.PublicKeys}

	info.Spec.PublicKeys = []apiv1.EncryptionKeySpec{}

	out := table.NewWriter(tables.Info, "", false, s.Output)
	out.Write(ClientServerVersion{
		Client: struct {
			Version bversion.Version `json:"version,omitempty"`
		}{Version: version.Get()},
		Server:    info.Spec,
		Namespace: ns,
	})
	return out.Err()
}
