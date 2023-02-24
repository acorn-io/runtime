package project

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/credentials"
	"github.com/acorn-io/acorn/pkg/hub"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
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
	p, err := c.ProjectDelete(ctx, lastPart(name))
	if err != nil {
		return nil, err
	}
	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return nil, err
	}
	if cfg.CurrentProject == name {
		cfg.CurrentProject = ""
		return p, cfg.Save()
	}
	return p, nil
}

func Exists(ctx context.Context, opts Options, name string) error {
	opts.Project = name
	c, err := Client(ctx, opts)
	if err != nil {
		return err
	}
	eg := errgroup.Group{}
	for _, projectName := range strings.Split(c.GetProject(), ",") {
		// local copy
		opts := opts
		opts.Project = projectName
		eg.Go(func() error {
			c, err = Client(ctx, opts)
			if err != nil {
				return err
			}
			_, err := c.ProjectGet(ctx, lastPart(opts.Project))
			return err
		})
	}
	return eg.Wait()
}

func timeoutProjectList(ctx context.Context, c client.Client) ([]apiv1.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.ProjectList(ctx)
}

func listLocalKubeconfig(ctx context.Context, wg *sync.WaitGroup, result chan<- listResult, opts Options) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		c, err := clientFromFile(opts.Kubeconfig, "", opts.ContextEnv)
		if err != nil {
			logrus.Debugf("local kubeconfig client ignored file=[%s] context=[%s]: %v", opts.Kubeconfig, opts.ContextEnv, err)
			// just ignore invalid clients
			return
		}

		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		projects, err := timeoutProjectList(ctx, c)
		cancel()
		result <- listResult{
			source: "local kubeconfig",
			err:    err,
			projects: typed.MapSlice(projects, func(project apiv1.Project) string {
				return project.Name
			}),
		}
	}()
}

type listResult struct {
	err      error
	source   string
	projects []string
}

func listHubServers(ctx context.Context, wg *sync.WaitGroup, creds *credentials.Store, cfg *config.CLIConfig, result chan<- listResult) {
	for _, hubServer := range cfg.HubServers {
		// copy for usage in goroutine
		hubServer := hubServer

		wg.Add(1)
		go func() {
			defer wg.Done()

			var projects []string
			cred, ok, err := creds.Get(ctx, hubServer)
			if err == nil && ok {
				subCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				projects, err = hub.Projects(subCtx, hubServer, cred.Password)
				cancel()
			}
			result <- listResult{
				source:   hubServer,
				err:      err,
				projects: projects,
			}
		}()
	}
}

func List(ctx context.Context, opts Options) (projects []string, warnings map[string]error, err error) {
	var (
		cfg = opts.CLIConfig
		// if the user sets --kubeconfig we only consider kubeconfig and no other source for listing
		onlyListLocalKubeconfig = opts.Kubeconfig != ""
	)

	if cfg == nil {
		cfg, err = config.ReadCLIConfig()
		if err != nil {
			return nil, nil, err
		}
	}

	creds, err := credentials.NewLocalOnlyStore(cfg)
	if err != nil {
		return nil, nil, err
	}

	var (
		wg     sync.WaitGroup
		result = make(chan listResult)
	)
	warnings = map[string]error{}

	listLocalKubeconfig(ctx, &wg, result, opts)
	if !onlyListLocalKubeconfig {
		listHubServers(ctx, &wg, creds, cfg, result)
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	for listResult := range result {
		if listResult.err != nil {
			warnings[listResult.source] = listResult.err
		}
		projects = append(projects, listResult.projects...)
	}

	sort.Strings(projects)
	return projects, warnings, nil
}
