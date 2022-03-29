package client

import (
	"context"
	"sort"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/k8sclient"
	"github.com/ibuildthecloud/herd/pkg/system"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type App struct {
	Name        string            `json:"name,omitempty"`
	Created     metav1.Time       `json:"created,omitempty"`
	Revision    string            `json:"revision,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Image   string             `json:"image,omitempty"`
	Volumes []v1.VolumeBinding `json:"volumes,omitempty"`
	Secrets []v1.SecretBinding `json:"secrets,omitempty"`

	Status v1.AppInstanceStatus `json:"status,omitempty"`
}

func Default() (Client, error) {
	k8sclient, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}
	ns := system.UserNamespace()
	return &client{
		Namespace: ns,
		Client:    k8sclient,
	}, nil
}

type Client interface {
	AppList(ctx context.Context) ([]App, error)
	//AppUpdate(ctx context.Context, app *App) error
	//AppCreate(ctx context.Context, app *App) (*App, error)
	//AppGet(ctx context.Context, name string) (*App, error)
	//AppStop(ctx context.Context, name string) error
	//AppStart(ctx context.Context, name string) error

	//Delete(ctx context.Context, obj any)
}

type client struct {
	Namespace string `json:"namespace,omitempty"`
	Client    kclient.WithWatch
}

func (c *client) appsForNS(ctx context.Context, eg *errgroup.Group, namespace string, result chan<- v1.AppInstance) {
	eg.Go(func() error {
		apps := &v1.AppInstanceList{}
		err := c.Client.List(ctx, apps, &kclient.ListOptions{
			Namespace: namespace,
		})
		if err != nil {
			return err
		}
		for _, app := range apps.Items {
			result <- app
			if app.Status.Namespace != "" {
				c.appsForNS(ctx, eg, app.Status.Namespace, result)
			}
		}
		return nil
	})
}

func waitAndClose[T any](eg *errgroup.Group, c chan T, err *error) {
	go func() {
		*err = eg.Wait()
		close(c)
	}()
}

func (c *client) AppList(ctx context.Context) (result []App, err error) {
	var (
		apps = make(chan v1.AppInstance)
		eg   = errgroup.Group{}
	)

	c.appsForNS(ctx, &eg, c.Namespace, apps)
	waitAndClose(&eg, apps, &err)

	for app := range apps {
		result = append(result, App{
			Name:        app.Name,
			Created:     app.CreationTimestamp,
			Revision:    app.ResourceVersion,
			Labels:      app.Labels,
			Annotations: app.Annotations,
			Image:       app.Spec.Image,
			Volumes:     app.Spec.Volumes,
			Secrets:     app.Spec.Secrets,
			Status:      app.Status,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return less(
			result[i].Name,
			result[j].Name,
			result[i].Status.Namespace,
			result[j].Status.Namespace,
		)
	})

	return
}

func less(terms ...string) bool {
	for i := range terms {
		if i%2 != 0 {
			continue
		}
		if terms[i] == terms[i+1] {
			continue
		}
		return terms[i] < terms[i+1]
	}
	return false
}
