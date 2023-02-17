package cli

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/acorn/pkg/version"
	bversion "github.com/acorn-io/baaah/pkg/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewInfo(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Info{client: c.ClientFactory}, cobra.Command{
		Use:          "info",
		SilenceUsage: true,
		Short:        "Info about acorn installation",
		Args:         cobra.NoArgs,
	})
	return cmd
}

type Info struct {
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o" default:"yaml"`
	client ClientFactory
}

type InfoCLIResponse struct {
	Client struct {
		Version bversion.Version  `json:"version,omitempty"`
		CLI     *config.CLIConfig `json:"cli,omitempty"`
	} `json:"client,omitempty"`
	Projects map[string]apiv1.InfoSpec `json:"projects,omitempty"`
}

func (s *Info) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	info, err := c.Info(cmd.Context())
	if err != nil {
		return err
	}

	// Testing/mocking ReadCLIConfig() is difficult. Any better way to test?
	cfg, err := config.ReadCLIConfig()
	if err != nil {
		logrus.Errorf("failed to read CLI config: %v", err)
		cfg = nil
	}
	projectInfo := make(map[string]apiv1.InfoSpec)

	// Format data from info response into map of project name to info response
	for _, subInfo := range info {
		projectInfo[subInfo.Name] = subInfo.Spec
	}

	out := table.NewWriter(tables.Info, false, s.Output)
	out.Write(InfoCLIResponse{
		Client: struct {
			Version bversion.Version  `json:"version,omitempty"`
			CLI     *config.CLIConfig `json:"cli,omitempty"`
		}{
			Version: version.Get(),
			CLI:     cfg.Sanitize(),
		},
		Projects: projectInfo,
	})
	return out.Err()
}
