package project

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
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

const (
	KubernetesProjectPrefix = "k8s.io"
	CustomProjectPrefix     = "custom"
)

var (
	csvSplit              = regexp.MustCompile(`\s*,\s*`)
	ErrNoCurrentProject   = errors.New("current project is not set")
	NoProjectMessageNoHub = "\n" +
		"\nAcorn CLI has not been initialized." +
		"\n" +
		"\nPlease run one of the following commands:" +
		"\n" +
		"\n  \"acorn install\"              Install acorn into a Kubernetes cluster." +
		"\n  \"acorn project use PROJECT\"  You already have a project but it's not set for some reason." +
		"\n                               Use \"acorn projects\" to find your project name."
	NoProjectMessage = "\n" +
		"\nAcorn CLI has not been initialized." +
		"\n" +
		"\nPlease run one of the following commands:" +
		"\n" +
		"\n  \"acorn login\"               (Easy mode:     Configure acorn to use cloud based resources. Fast and easy.)" +
		"\n  \"acorn install\"             (Hard mode:     Install acorn into a Kubernetes cluster. There be dragons.)" +
		"\n  \"acorn project use PROJECT\" (Confused mode: You already have a project but it's not set for some reason." +
		"\n                                              Use \"acorn projects\" to find your project name.)"
)

type Options struct {
	Project     string
	Kubeconfig  string
	Context     string
	Namespace   string
	AllProjects bool
}

func Client(ctx context.Context, opts Options) (client.Client, error) {
	c, err := lookup(ctx, opts)
	if err != nil {
		return nil, err
	}

	if c == nil {
		return nil, ErrNoCurrentProject
	}
	return c, nil
}

func lookup(ctx context.Context, opts Options) (client.Client, error) {
	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return nil, err
	}

	if opts.Kubeconfig != "" {
		return clientFromFile(opts.Kubeconfig, opts)
	}

	projects, err := getDesiredProjects(ctx, cfg, opts)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		c := defaultK8s(ctx, opts)
		if c == nil {
			return nil, nil
		}
		defaultProject := c.GetProject()
		if opts.AllProjects {
			defaultProject = ""
		}
		return client.NewMultiClient(defaultProject, c.GetNamespace(), &projectClientFactory{
			defaultProject: defaultProject,
			clients:        []client.Client{c},
			cfg:            cfg,
		}), nil
	}

	var clients []client.Client
	for _, project := range projects {
		client, err := getClient(ctx, cfg.Kubeconfigs, cfg.HubServers, project, opts.Namespace, opts.Context)
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
	}), nil
}

type projectClientFactory struct {
	defaultProject string
	clients        []client.Client
	cfg            *config.CLIConfig
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

	return getClient(ctx, p.cfg.Kubeconfigs, p.cfg.HubServers, project, "", "")
}

func (p *projectClientFactory) List(ctx context.Context) ([]client.Client, error) {
	return p.clients, nil
}

func (p *projectClientFactory) DefaultProject() string {
	return p.defaultProject
}

func getNamespace(opts Options) string {
	if opts.Namespace != "" {
		return opts.Namespace
	}
	return system.UserNamespace()
}

func clientFromFile(kubeconfig string, opts Options) (client.Client, error) {
	cfg, err := restconfig.FromFile(kubeconfig, opts.Context)
	if err != nil {
		return nil, err
	}
	ns := getNamespace(opts)
	return client.New(cfg, CustomProjectPrefix+"-"+filepath.Base(kubeconfig)+"/"+ns, ns)
}

func defaultK8s(ctx context.Context, opts Options) client.Client {
	ns := getNamespace(opts)
	c, err := getClient(ctx, nil, nil, KubernetesProjectPrefix+"/"+ns, "", opts.Context)
	if err != nil {
		return nil
	}
	_, err = c.Info(ctx)
	if err != nil {
		return nil
	}
	return c
}

func getClient(ctx context.Context, kubeconfigs map[string]string, hubServers []string, project, nsOverride, contextOverride string) (client.Client, error) {
	server, ns, ok := strings.Cut(project, "/")
	if !ok {
		return nil, fmt.Errorf("invalid format for project string, should contain '/': %s", project)
	}
	if server == KubernetesProjectPrefix {
		restConfig, err := restconfig.Default()
		if err != nil {
			return nil, err
		}
		if nsOverride != "" {
			ns = nsOverride
		}
		return client.New(restConfig, project, ns)
	}

	if kubeconfig := kubeconfigs[server]; kubeconfig != "" {
		config, err := restconfig.FromFile(kubeconfig, contextOverride)
		if err != nil {
			return nil, err
		}
		return client.New(config, project, ns)
	}

	credStore, err := credentials.NewStore(nil)
	if err != nil {
		return nil, err
	}

	for _, hubServer := range hubServers {
		if hubServer != server {
			continue
		}

		cred, ok, err := credStore.Get(ctx, hubServer)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, fmt.Errorf("failed to find authentication token for server %s,"+
				" please run 'acorn login %s' first", hubServers, hubServers)
		}

		url, ns, err := hub.ProjectURLAndNamespace(ctx, project, cred.Password)
		if err != nil {
			return nil, err
		}
		return client.New(&rest.Config{
			Host:        url,
			BearerToken: cred.Password,
		}, project, ns)
	}

	return nil, fmt.Errorf("failed to find connection information for project %s,"+
		" you might need to run 'acorn login %s' first", project, server)
}

func getDesiredProjects(ctx context.Context, cfg *config.CLIConfig, opts Options) (result []string, err error) {
	if opts.AllProjects {
		return List(ctx, cfg, opts)
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
