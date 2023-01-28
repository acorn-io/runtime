package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/credentials"
	"github.com/acorn-io/acorn/pkg/hub"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/adrg/xdg"
	"k8s.io/client-go/rest"
)

var (
	csvSplit              = regexp.MustCompile(`\s*,\s*`)
	ErrNoKubernetesConfig = errors.New("no kubeconfig file found try creating one at $HOME/.kube/config")
	NoProjectMessageNoHub = "\n" +
		"\nA valid Acorn client configuration can not be found." +
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
	Project       string
	Kubeconfig    string
	KubeconfigEnv string
	ContextEnv    string
	NamespaceEnv  string
	AllProjects   bool
	CLIConfig     *config.CLIConfig
}

func (o Options) WithCLIConfig(cfg *config.CLIConfig) Options {
	o.CLIConfig = cfg
	return o
}

func Client(ctx context.Context, opts Options) (client.Client, error) {
	return lookup(ctx, opts)
}

func localProjectToNamespace(project string) (string, error) {
	parts := strings.Split(project, "/")
	if len(parts) > 1 {
		return "", fmt.Errorf("failed to determine target project, / is not allowed in project [%s] name when using local kubeconfig", project)
	}
	return parts[0], nil
}

func lookupNamespaceForLocal(opts Options, fromKubeconfig string) (string, error) {
	if opts.AllProjects {
		return "", nil
	}
	if opts.Project != "" {
		return localProjectToNamespace(opts.Project)
	}
	if opts.CLIConfig.CurrentProject != "" {
		if strings.Contains(opts.CLIConfig.CurrentProject, "/") {
			return system.DefaultUserNamespace, nil
		}
		return opts.CLIConfig.CurrentProject, nil
	}
	if opts.NamespaceEnv != "" {
		return localProjectToNamespace(opts.NamespaceEnv)
	}
	if fromKubeconfig != "" {
		return fromKubeconfig, nil
	}
	return system.DefaultUserNamespace, nil
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

	// If user specifies --kubeconfig them always read from it regardless of project config.  If the current
	// project contains a "/" then fail
	if opts.Kubeconfig != "" {
		return clientFromFile(opts.Kubeconfig, opts)
	}

	projects, err := getDesiredProjects(ctx, cfg, opts)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		c, err := noConfigClient(ctx, opts)
		if err != nil {
			return nil, err
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

func lookupKubeconfig(opts Options) (string, bool) {
	if opts.Kubeconfig != "" {
		return opts.Kubeconfig, true
	}
	if opts.KubeconfigEnv != "" {
		return opts.KubeconfigEnv, true
	}
	homeConfig := filepath.Join(xdg.Home, ".kube", "config")
	if _, err := os.Stat(homeConfig); err == nil {
		return homeConfig, true
	}
	return "", false
}

func clientFromFile(kubeconfig string, opts Options) (client.Client, error) {
	clientConfig := restconfig.ClientConfigFromFile(kubeconfig, opts.ContextEnv)
	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	def, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, err
	}
	// assume if the result is "default" the user didn't set one
	if def == "default" {
		def = ""
	}
	ns, err := lookupNamespaceForLocal(opts, def)
	if err != nil {
		return nil, err
	}
	return client.New(cfg, ns, ns)
}

// noConfigClient will try to find acorn locally installed in a local k8s, if it fails in anyway,
// ignore it and continue
func noConfigClient(ctx context.Context, opts Options) (client.Client, error) {
	result, found := lookupKubeconfig(opts)
	if !found {
		return nil, ErrNoKubernetesConfig
	}
	return clientFromFile(result, opts)
}

func ParseProject(project string, kubeconfigs map[string]string) (server, account, namespace string, err error) {
	parts := strings.Split(project, "/")
	if len(parts) == 0 || len(parts) > 3 {
		return "", "", "", fmt.Errorf("invalid project name [%s]: must contain zero, one or two slashes [/]", project)
	}
	switch len(parts) {
	case 1:
		if strings.Contains(parts[0], ".") {
			return "", "", "", fmt.Errorf("invalid project name [%s]: can not contain \".\"", project)
		}
		if strings.Contains(parts[0], ":") {
			return "", "", "", fmt.Errorf("invalid project name [%s]: can not contain \":\"", project)
		}
		return "", "", parts[0], nil
	case 2:
		if strings.Contains(parts[0], ".") {
			return "", "", "", fmt.Errorf("invalid project name [%s]: part before / can not contain \".\" unless there are three parts (ex: acorn.io/account/name)", project)
		}
		if strings.Contains(parts[0], ":") {
			return "", "", "", fmt.Errorf("invalid project name [%s]: part before / can not contain \":\" unless there are three parts (ex: acorn.io/account/name)", project)
		}
		if kubeconfig := kubeconfigs[parts[0]]; kubeconfig != "" {
			return "", parts[0], parts[1], nil
		}
		return system.DefaultHubAddress, parts[0], parts[1], nil
	case 3:
		return parts[0], parts[1], parts[2], nil
	}
	panic(fmt.Sprintf("unreachable: parts len of %d not handled in %s", len(parts), project))
}

func getClient(ctx context.Context, cfg *config.CLIConfig, opts Options, project string) (client.Client, error) {
	server, account, namespace, err := ParseProject(project, cfg.Kubeconfigs)
	if err != nil {
		return nil, err
	}

	if server == "" {
		if account != "" {
			if kubeconfig := cfg.Kubeconfigs[account]; kubeconfig == "" {
				return nil, fmt.Errorf("failed to find kubeconfig for %s", account)
			} else {
				config, err := restconfig.FromFile(kubeconfig, opts.ContextEnv)
				if err != nil {
					return nil, err
				}
				return client.New(config, project, namespace)
			}
		}

		cfgFile, ok := lookupKubeconfig(opts)
		if !ok {
			return nil, ErrNoKubernetesConfig
		}
		c, err := restconfig.FromFile(cfgFile, opts.ContextEnv)
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
			url := cfg.TestProjectURLs[server+"/"+account]
			if url == "" {
				url, err = hub.ProjectURL(ctx, server, account, cred.Password)
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

func getDesiredProjects(ctx context.Context, cfg *config.CLIConfig, opts Options) (result []string, err error) {
	if opts.AllProjects {
		return List(ctx, opts.WithCLIConfig(cfg))
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
