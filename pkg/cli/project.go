package cli

import (
	"sort"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewProject(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Project{client: c.ClientFactory}, cobra.Command{
		Use:     "project [flags]",
		Aliases: []string{"projects"},
		Example: `
acorn project`,
		SilenceUsage:      true,
		Short:             "Manage projects",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, projectsCompletion).complete,
	})
	cmd.AddCommand(NewProjectCreate(c))
	cmd.AddCommand(NewProjectRm(c))
	cmd.AddCommand(NewProjectUse(c))
	cmd.AddCommand(NewProjectUpdate(c))
	return cmd
}

type Project struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

type projectEntry struct {
	Name          string   `json:"name,omitempty"`
	Default       bool     `json:"default,omitempty"`
	Regions       []string `json:"regions,omitempty"`
	DefaultRegion string   `json:"default-region,omitempty"`
}

func (a *Project) Run(cmd *cobra.Command, args []string) error {
	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return err
	}

	var projectNames []string
	if len(args) == 1 {
		err := project.Exists(cmd.Context(), a.client.Options().WithCLIConfig(cfg), args[0])
		if err != nil {
			return err
		}
		projectNames = append(projectNames, args[0])
	} else {
		projects, warnings, err := project.List(cmd.Context(), a.client.Options().WithCLIConfig(cfg))
		if err != nil {
			return err
		}
		if len(args) == 0 {
			projectNames = append(projectNames, projects...)
		} else {
			for _, arg := range args {
				if slices.Contains(projects, arg) {
					projectNames = append(projectNames, arg)
				}
			}
		}
		for _, env := range typed.SortedKeys(warnings) {
			logrus.Warnf("Could not list projects from [%s]: %v", env, warnings[env])
		}
	}

	defaultProject := cfg.CurrentProject

	c, err := project.Client(cmd.Context(), a.client.Options())
	if err == nil {
		defaultProject = c.GetProject()
	}

	out := table.NewWriter(tables.ProjectClient, a.Quiet, a.Output)
	projectDetails, err := project.GetDetails(cmd.Context(), a.client.Options(), projectNames)
	if err != nil {
		return err
	}
	sort.Slice(projectDetails, func(i, j int) bool {
		return projectDetails[i].FullName < projectDetails[j].FullName
	})

	for _, projectItem := range projectDetails {
		if projectItem.Err != nil {
			logrus.Warnf("Could not list details of project [%s]: %v", projectItem.FullName, projectItem.Err)
			continue
		}
		if projectItem.Project != nil {
			supportedRegions := projectItem.Project.GetSupportedRegions()
			defaultRegion := projectItem.Project.GetRegion()
			for i, supportedRegion := range supportedRegions {
				if supportedRegion == defaultRegion {
					supportedRegions[i] = supportedRegion + "*"
				}
			}
			out.WriteFormatted(projectEntry{
				Name:    projectItem.FullName,
				Default: defaultProject == projectItem.FullName,
				Regions: supportedRegions,
			}, projectItem.Project)
		}
	}

	return out.Err()
}
