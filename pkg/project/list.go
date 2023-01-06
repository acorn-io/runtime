package project

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/credentials"
	"github.com/acorn-io/acorn/pkg/hub"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/sirupsen/logrus"
)

func List(ctx context.Context, cfg *config.CLIConfig, opts Options) (result []string, err error) {
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
			return CustomProjectPrefix + "/" + project.Name
		}), nil
	}

	for _, key := range typed.SortedKeys(cfg.Kubeconfigs) {
		c, err := getClient(ctx, cfg.Kubeconfigs, nil, key+"/", "", "")
		if err == nil {
			projs, err := c.ProjectList(ctx)
			if err == nil {
				result = append(result, typed.MapSlice(projs, func(project apiv1.Project) string {
					return key + "/" + project.Name
				})...)
			} else {
				logrus.Debugf("failed to list projects in k8s name config %s: %v", key, err)
			}
		} else {
			logrus.Debugf("failed to get client for k8s named config %s: %v", key, err)
		}
	}

	creds, err := credentials.NewStore(nil)
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

	c := defaultK8s(ctx, opts)
	if c != nil {
		projs, err := c.ProjectList(ctx)
		if err == nil {
			result = append(result, typed.MapSlice(projs, func(project apiv1.Project) string {
				return KubernetesProjectPrefix + "/" + project.Name
			})...)
		} else {
			logrus.Debugf("failed to list projects in default k8s context: %v", err)
		}
	}

	return
}
