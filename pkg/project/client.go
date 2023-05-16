package project

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/credentials"
	"github.com/acorn-io/acorn/pkg/hub"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"k8s.io/client-go/rest"
)

var (
	csvSplit = regexp.MustCompile(`\s*,\s*`)
)

type Options struct {
	Project     string
	Kubeconfig  string
	ContextEnv  string
	AllProjects bool
	CLIConfig   *config.CLIConfig
}

func (o Options) WithCLIConfig(cfg *config.CLIConfig) Options {
	o.CLIConfig = cfg
	return o
}

func Client(ctx context.Context, opts Options) (client.Client, error) {
	return lookup(ctx, opts)
}

func lookup(ctx context.Context, opts Options) (client.Client, error) {
	var (
		cfg = opts.CLIConfig
		err error
	)

	if cfg == nil {
		cfg, err = config.ReadCLIConfig()
		if err != nil {
			return nil, err
		}
		opts.CLIConfig = cfg
	}

	projects, err := getDesiredProjects(ctx, cfg, opts)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		projects = []string{system.DefaultUserNamespace}
	}

	var clients []client.Client
	for _, project := range projects {
		client, err := getClient(ctx, cfg, opts, project)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}

	defaultProject := ""
	if len(clients) == 1 {
		defaultProject = clients[0].GetProject()
	}

	return client.NewMultiClient(strings.Join(projects, ","), "", &projectClientFactory{
		defaultProject: defaultProject,
		clients:        clients,
		cfg:            cfg,
		opts:           opts,
	}), nil
}

type projectClientFactory struct {
	defaultProject string
	clients        []client.Client
	cfg            *config.CLIConfig
	opts           Options
}

func (p *projectClientFactory) ForProject(ctx context.Context, project string) (client.Client, error) {
	if project == "" {
		return p.clients[0], nil
	}
	for _, client := range p.clients {
		if client.GetProject() == project {
			return client, nil
		}
	}

	return getClient(ctx, p.cfg, p.opts, project)
}

func (p *projectClientFactory) List(ctx context.Context) ([]client.Client, error) {
	return p.clients, nil
}

func (p *projectClientFactory) DefaultProject() string {
	return p.defaultProject
}

func clientFromFile(kubeconfig string, ns string, context string) (client.Client, error) {
	cfg, err := restconfig.FromFile(kubeconfig, context)
	if err != nil {
		return nil, err
	}
	return client.New(cfg, ns, ns)
}

func ParseProject(project string, kubeconfigs map[string]string) (serverOrKubeconfig, account, namespace string, isKubeconfig bool, err error) {
	parts := strings.Split(project, "/")
	if len(parts) == 0 || len(parts) > 3 {
		return "", "", "", false, fmt.Errorf("invalid project name [%s]: must contain zero, one or two slashes [/]", project)
	}
	switch len(parts) {
	case 1:
		if strings.Contains(parts[0], ".") {
			return "", "", "", false, fmt.Errorf("invalid project name [%s]: can not contain \".\"", project)
		}
		if strings.Contains(parts[0], ":") {
			return "", "", "", false, fmt.Errorf("invalid project name [%s]: can not contain \":\"", project)
		}
		return "", "", parts[0], true, nil
	case 2:
		if strings.Contains(parts[0], ".") {
			return "", "", "", false, fmt.Errorf("invalid project name [%s]: part before / can not contain \".\" unless there are three parts (ex: acorn.io/account/name)", project)
		}
		if strings.Contains(parts[0], ":") {
			return "", "", "", false, fmt.Errorf("invalid project name [%s]: part before / can not contain \":\" unless there are three parts (ex: acorn.io/account/name)", project)
		}
		if kubeconfig := kubeconfigs[parts[0]]; kubeconfig != "" {
			return kubeconfig, parts[0], parts[1], true, nil
		}
		return system.DefaultHubAddress, parts[0], parts[1], false, nil
	case 3:
		return parts[0], parts[1], parts[2], false, nil
	}
	panic(fmt.Sprintf("unreachable: parts len of %d not handled in %s", len(parts), project))
}

func getDesiredProjects(ctx context.Context, cfg *config.CLIConfig, opts Options) (result []string, err error) {
	if opts.AllProjects {
		projects, _, err := List(ctx, opts.WithCLIConfig(cfg))
		return projects, err
	}
	p := strings.TrimSpace(opts.Project)
	if p == "" {
		p = strings.TrimSpace(cfg.CurrentProject)
	}
	if strings.TrimSpace(p) == "" {
		return nil, nil
	}
	for _, project := range csvSplit.Split(p, -1) {
		if target := cfg.ProjectAliases[project]; target == "" {
			result = append(result, project)
		} else {
			result = append(result, csvSplit.Split(strings.TrimSpace(target), -1)...)
		}
	}
	return result, nil
}

func getClient(ctx context.Context, cfg *config.CLIConfig, opts Options, project string) (client.Client, error) {
	serverOrKubeconfig, account, namespace, isKubeconfig, err := ParseProject(project, cfg.Kubeconfigs)
	if err != nil {
		return nil, err
	}

	if isKubeconfig {
		if serverOrKubeconfig == "" {
			c, err := restconfig.FromFile(opts.Kubeconfig, opts.ContextEnv)
			if err != nil {
				return nil, err
			}
			return client.New(c, project, namespace)
		}

		config, err := restconfig.FromFile(serverOrKubeconfig, opts.ContextEnv)
		if err != nil {
			return nil, err
		}
		return client.New(config, project, namespace)
	}

	credStore, err := credentials.NewStore(cfg, nil)
	if err != nil {
		return nil, err
	}

	cred, ok, err := credStore.Get(ctx, serverOrKubeconfig)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("failed to find authentication token for server %s,"+
			" please run 'acorn login %s' first", serverOrKubeconfig, serverOrKubeconfig)
	}

	return &client.DeferredClient{
		Project:   project,
		Namespace: namespace,
		New: func() (client.Client, error) {
			url := cfg.TestProjectURLs[serverOrKubeconfig+"/"+account]
			if url == "" {
				url, err = hub.ProjectURL(ctx, serverOrKubeconfig, account, cred.Password)
				if err != nil {
					return nil, err
				}
			}
			return client.New(&rest.Config{
				Host:        url,
				BearerToken: cred.Password,
			}, project, namespace)
		},
	}, nil
}
