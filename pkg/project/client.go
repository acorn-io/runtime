package project

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/manager"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/sirupsen/logrus"
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
}

func (o Options) CLIConfig() (*config.CLIConfig, error) {
	return config.ReadCLIConfig(o.Kubeconfig != "")
}

func Client(ctx context.Context, opts Options) (client.Client, error) {
	return lookup(ctx, opts)
}

func lookup(ctx context.Context, opts Options) (client.Client, error) {
	cfg, err := opts.CLIConfig()
	if err != nil {
		return nil, err
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

func RenderProjectName(project string, defaultContext string) string {
	defaultServer, defaultAccount, hasDefaultContext := strings.Cut(defaultContext, "/")
	if !hasDefaultContext {
		defaultServer = config.LocalServer
		defaultAccount = ""
	}

	for _, prefix := range []string{defaultServer + "/" + defaultAccount + "/", defaultServer + "/"} {
		if strings.HasPrefix(project, prefix) {
			return strings.TrimPrefix(project, prefix)
		}
	}

	return project
}

func ParseProject(project string, defaultContext string) (server, account, namespace string, isKubeconfig bool, err error) {
	defaultServer, defaultAccount, hasDefaultContext := strings.Cut(defaultContext, "/")
	if !hasDefaultContext {
		if defaultContext != "" && defaultContext != config.LocalServer {
			logrus.Errorf("Invalid default context set in the CLI config [%s], assuming to [%s/]", defaultContext, config.LocalServer)
		}
		defaultServer = config.LocalServer
		defaultAccount = ""
	}

	parts := strings.Split(project, "/")
	if len(parts) < 3 {
		switch len(parts) {
		case 1:
			parts = []string{defaultServer, defaultAccount, parts[0]}
		case 2:
			if parts[0] == config.LocalServer {
				parts = []string{config.LocalServer, "", parts[1]}
			} else {
				parts = []string{defaultServer, parts[0], parts[1]}
			}
		}
	} else if len(parts) != 3 {
		return "", "", "", false, fmt.Errorf("invalid project name [%s]: can not contain more that two \"/\"", project)
	}
	if strings.Contains(parts[2], ".") {
		return "", "", "", false, fmt.Errorf("invalid project name [%s]: can not contain \".\"", parts[2])
	}
	if parts[0] == config.LocalServer && parts[1] != "" {
		return "", "", "", false, fmt.Errorf("invalid project name [%s]: account can not be set for local kubeconfig", project)
	}
	return parts[0], parts[1], parts[2], parts[0] == config.LocalServer, nil
}

func getDesiredProjects(ctx context.Context, cfg *config.CLIConfig, opts Options) (result []string, err error) {
	if opts.AllProjects {
		projects, _, err := List(ctx, true, opts)
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
	server, account, namespace, isKubeconfig, err := ParseProject(project, cfg.DefaultContext)
	if err != nil {
		return nil, err
	}

	if isKubeconfig {
		c, err := restconfig.FromFile(opts.Kubeconfig, opts.ContextEnv)
		if err != nil {
			return nil, err
		}
		return client.New(c, project, namespace)
	}

	credStore, err := credentials.NewStore(cfg, nil)
	if err != nil {
		return nil, err
	}

	cred, ok, err := credStore.Get(ctx, server)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("failed to find authentication token for server %s,"+
			" please run 'acorn login %s' first", server, server)
	}

	return &client.DeferredClient{
		Project:   project,
		Namespace: namespace,
		New: func() (client.Client, error) {
			url := cfg.ProjectURLs[server+"/"+account]
			if url == "" {
				url, err = manager.ProjectURL(ctx, server, account, cred.Password)
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
