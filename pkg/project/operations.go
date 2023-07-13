package project

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/typed"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/manager"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func lastPart(s string) string {
	parts := strings.Split(s, "/")
	return parts[len(parts)-1]
}

func Create(ctx context.Context, opts Options, name, defaultRegion string, supportedRegions []string) error {
	opts.Project = name
	c, err := Client(ctx, opts)
	if err != nil {
		return err
	}
	_, err = c.ProjectCreate(ctx, lastPart(name), defaultRegion, supportedRegions)
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

func Update(ctx context.Context, opts Options, project DetailProject, defaultRegion string, supportedRegions []string) error {
	opts.Project = project.FullName
	c, err := Client(ctx, opts)
	if err != nil {
		return err
	}
	_, err = c.ProjectUpdate(ctx, project.Project, defaultRegion, supportedRegions)
	return err
}

func timeoutProjectList(ctx context.Context, c client.Client) ([]apiv1.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.ProjectList(ctx)
}

func listLocalKubeconfig(ctx context.Context, opts Options) listResult {
	c, err := clientFromFile(opts.Kubeconfig, "", opts.ContextEnv)
	if err != nil {
		logrus.Debugf("local kubeconfig client ignored file=[%s] context=[%s]: %v", opts.Kubeconfig, opts.ContextEnv, err)
		// just ignore invalid clients
		return listResult{}
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	projects, err := timeoutProjectList(ctx, c)
	cancel()
	return listResult{
		source: "local kubeconfig",
		err:    err,
		projects: typed.MapSlice(projects, func(project apiv1.Project) string {
			return project.Name
		}),
	}
}

type listResult struct {
	err      error
	source   string
	projects []string
}

type DetailProject struct {
	FullName string
	Project  *apiv1.Project
	Err      error
}

func listAcornServer(ctx context.Context, server string, creds *credentials.Store) listResult {
	var projects []string
	cred, ok, err := creds.Get(ctx, server)
	if err == nil && ok {
		subCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		projects, err = manager.Projects(subCtx, server, cred.Password)
		cancel()
	}
	return listResult{
		source:   server,
		err:      err,
		projects: projects,
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

	managerHost, _, managerHostExists := strings.Cut(cfg.CurrentProject, "/")

	creds, err := credentials.NewLocalOnlyStore(cfg)
	if err != nil {
		return nil, nil, err
	}

	var result listResult
	warnings = map[string]error{}

	if onlyListLocalKubeconfig || !managerHostExists {
		result = listLocalKubeconfig(ctx, opts)
	} else {
		result = listAcornServer(ctx, managerHost, creds)
	}

	if result.err != nil {
		warnings[result.source] = result.err
	}
	projects = append(projects, result.projects...)

	sort.Strings(projects)
	return projects, warnings, nil
}

func GetDetails(ctx context.Context, opts Options, projectNames []string) (projects []DetailProject, err error) {
	var (
		wg     sync.WaitGroup
		result = make(chan DetailProject)
	)

	err = getProjectDetails(ctx, &wg, result, opts, projectNames)
	if err != nil {
		return nil, err
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	for detailProject := range result {
		projects = append(projects, detailProject)
	}

	return projects, nil
}

func getProjectDetails(ctx context.Context, wg *sync.WaitGroup, result chan<- DetailProject, opts Options, projectNames []string) error {
	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return err
	}
	for _, projectName := range projectNames {
		wg.Add(1)
		go func(projectName string, opts Options) {
			defer wg.Done()
			// Launch a goroutine for each project to retrieve its information
			opts.Project = projectName
			c, err := getClient(ctx, cfg, opts, projectName)
			if err != nil {
				logrus.Warnf("unable to get client for %s: %s", projectName, err)
				// just ignore invalid clients
				return
			}
			project, err := c.ProjectGet(ctx, lastPart(opts.Project))
			result <- DetailProject{
				FullName: projectName,
				Project:  project,
				Err:      err,
			}
		}(projectName, opts)
	}
	return nil
}
