package cli

import (
	"encoding/json"
	"reflect"

	"github.com/acorn-io/baaah/pkg/typed"
	bversion "github.com/acorn-io/baaah/pkg/version"
	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/tables"
	"github.com/acorn-io/runtime/pkg/version"
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
	Projects map[string]any `json:"projects,omitempty"`
}

func (s *Info) Run(cmd *cobra.Command, _ []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	info, err := c.Info(cmd.Context())
	if err != nil {
		return err
	}

	// Testing/mocking ReadCLIConfig() is difficult. Any better way to test?
	cfg, err := config.ReadCLIConfig(s.client.AcornConfigFile(), true)
	if err != nil {
		logrus.Errorf("failed to read CLI config: %v", err)
		cfg = nil
	}

	projectInfo := make(map[string]any, len(info))
	// Format data from info response into map of project name to info response
	for _, subInfo := range info {
		// Remove unset fields in the config and userConfig
		regionInfo := make(map[string]any, len(subInfo.Regions)+1)
		if len(subInfo.ExtraData) > 0 {
			regionInfo["extraData"] = subInfo.ExtraData
		}
		for region, spec := range subInfo.Regions {
			specMap, err := removeUnsetFields(spec, "config", "userConfig")
			if err != nil {
				return err
			}
			regionInfo[region] = specMap
		}

		projectInfo[subInfo.Name] = regionInfo
	}

	out := table.NewWriter(tables.Info, false, s.Output)
	out.WriteFormatted(InfoCLIResponse{
		Client: struct {
			Version bversion.Version  `json:"version,omitempty"`
			CLI     *config.CLIConfig `json:"cli,omitempty"`
		}{
			Version: version.Get(),
			CLI:     cfg.Sanitize(),
		},
		Projects: projectInfo,
	}, nil)
	// This is somewhat of a hack that forces this single item to print
	if err := out.Flush(); err != nil {
		return err
	}
	return out.Err()
}

func removeUnsetFields(spec v1.InfoSpec, configKeys ...string) (map[string]any, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	var specMap map[string]any
	if err := json.Unmarshal(b, &specMap); err != nil {
		return nil, err
	}

	for _, key := range configKeys {
		if cfg, ok := specMap[key].(map[string]any); ok {
			for _, entry := range typed.Sorted(cfg) {
				if entry.Value == nil || reflect.ValueOf(entry.Value).IsZero() {
					delete(cfg, entry.Key)
				}
			}
			if len(cfg) == 0 {
				delete(specMap, key)
			}
		}
	}

	return specMap, nil
}
