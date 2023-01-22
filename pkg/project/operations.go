package project

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/credentials"
	"github.com/acorn-io/acorn/pkg/hub"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/sirupsen/logrus"
)

func lastPart(s string) string {
	parts := strings.Split(s, "/")
	return parts[len(parts)-1]
}

func Create(ctx context.Context, opts Options, name string) error {
	opts.Project = name
	c, err := Client(ctx, opts)
	if err != nil {
		return err
	}
	_, err = c.ProjectCreate(ctx, lastPart(name))
	return err
}

func Remove(ctx context.Context, opts Options, name string) (*apiv1.Project, error) {
	opts.Project = name
	c, err := Client(ctx, opts)
	if err != nil {
		return nil, err
	}
	return c.ProjectDelete(ctx, lastPart(name))
}

func Get(ctx context.Context, opts Options, name string) (project *apiv1.Project, err error) {
	opts.Project = name
	c, err := Client(ctx, opts)
	if err != nil {
		return nil, err
	}
	return c.ProjectGet(ctx, lastPart(name))
}

func List(ctx context.Context, opts Options) (result []string, err error) {
	cfg := opts.CLIConfig
	if cfg == nil {
		cfg, err = config.ReadCLIConfig()
		if err != nil {
			return nil, err
		}
	}

	// if the user sets --kubeconfig we only consider kubeconfig and nothing else
	if opts.Kubeconfig != "" {
		c, err := clientFromFile(opts.Kubeconfig, opts)
		if err != nil {
			return nil, err
		}
		projs, err := c.ProjectList(ctx)
		if err != nil {
			return nil, err
		}
		return typed.MapSlice(projs, func(project apiv1.Project) string {
			return project.Name
		}), nil
	}

	for _, key := range typed.SortedKeys(cfg.Kubeconfigs) {
		kubeconfig := cfg.Kubeconfigs[key]
		cfg, err := restconfig.FromFile(kubeconfig, opts.ContextEnv)
		if err != nil {
			logrus.Debugf("failed to load kubeconfig [%s]: %v", kubeconfig, err)
			continue
		}

		c, err := client.New(cfg, "", "")
		if err != nil {
			logrus.Debugf("failed to build client for kubeconfig [%s]: %v", kubeconfig, err)
			continue
		}

		projs, err := c.ProjectList(ctx)
		if err != nil {
			logrus.Debugf("failed to list projects for kubeconfig [%s]: %v", kubeconfig, err)
			continue
		}

		result = append(result, typed.MapSlice(projs, func(project apiv1.Project) string {
			return key + "/" + project.Name
		})...)
	}

	creds, err := credentials.NewStore(cfg, nil)
	if err != nil {
		return nil, err
	}

	for _, hubServer := range cfg.HubServers {
		if _, ok := cfg.Kubeconfigs[hubServer]; ok {
			continue
		}
		cred, ok, err := creds.Get(ctx, hubServer)
		if err == nil && ok {
			projects, err := hub.Projects(ctx, hubServer, cred.Password)
			if err == nil {
				result = append(result, projects...)
			} else {
				logrus.Debugf("failed to list projects in hub server %s: %v", hubServer, err)
			}
		} else if err != nil {
			logrus.Debugf("failed to get cred for hub server %s: %v", hubServer, err)
		}
	}

	c := noConfigClient(ctx, opts)
	if c != nil {
		projects, err := c.ProjectList(ctx)
		if err == nil {
			result = append(result, typed.MapSlice(projects, func(project apiv1.Project) string {
				return project.Name
			})...)
		} else {
			logrus.Debugf("failed to list projects in default k8s context: %v", err)
		}
	}

	return
}
