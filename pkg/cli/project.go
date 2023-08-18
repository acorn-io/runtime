package cli

import (
	"fmt"
	"sort"

	"github.com/acorn-io/baaah/pkg/typed"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/project"
	"github.com/acorn-io/runtime/pkg/tables"
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
		ValidArgsFunction: newCompletion(c.ClientFactory, projectsCompletion(c.ClientFactory)).complete,
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
	var projectNames []string
	if len(args) == 1 {
		err := project.Exists(cmd.Context(), a.client.Options(), args[0])
		if err != nil {
			return err
		}
		projectNames = append(projectNames, args[0])
	} else {
		projects, warnings, err := project.List(cmd.Context(), false, a.client.Options())
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
			if env == config.LocalServerEnv && a.client.Options().Kubeconfig == "" {
				continue
			}
			logrus.Warnf("Could not list projects from [%s]: %v", env, warnings[env])
		}
	}

	cfg, err := a.client.Options().CLIConfig()
	if err != nil {
		return err
	}

	defaultProject := project.RenderProjectName(cfg.CurrentProject, cfg.DefaultContext)

	c, err := project.Client(cmd.Context(), a.client.Options())
	if err == nil {
		defaultProject = project.RenderProjectName(c.GetProject(), cfg.DefaultContext)
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

		projectName := project.RenderProjectName(projectItem.FullName, cfg.DefaultContext)
		if projectItem.Project != nil {
			if projectItem.Project.Annotations == nil {
				projectItem.Project.Annotations = map[string]string{}
			}
			projectItem.Project.Annotations["project-name"] = projectName
			projectItem.Project.Annotations["default-project"] = fmt.Sprint(defaultProject == projectName)

			supportedRegions := projectItem.Project.Status.SupportedRegions
			defaultRegion := projectItem.Project.Status.DefaultRegion
			if len(supportedRegions) > 1 {
				for i, supportedRegion := range supportedRegions {
					if supportedRegion == defaultRegion {
						supportedRegions[i] = supportedRegion + "*"
					}
				}
			}

			out.WriteFormatted(projectEntry{
				Name:    projectName,
				Default: defaultProject == projectName,
				Regions: supportedRegions,
			}, projectItem.Project)
		}
	}

	return out.Err()
}
