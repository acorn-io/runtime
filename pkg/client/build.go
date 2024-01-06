package client

import (
	"context"
	"os"
	"path/filepath"

	"github.com/acorn-io/aml"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/buildclient"
	"github.com/acorn-io/runtime/pkg/digest"
	"github.com/acorn-io/runtime/pkg/vcs"
	"github.com/denisbrodbeck/machineid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) AcornImageBuildDelete(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	builder, err := c.AcornImageBuildGet(ctx, name)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}

	return builder, c.Client.Delete(ctx, builder)
}

func (c *DefaultClient) AcornImageBuildGet(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	builder := &apiv1.AcornImageBuild{}
	return builder, c.Client.Get(ctx, kclient.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, builder)
}

func (c *DefaultClient) AcornImageBuildList(ctx context.Context) ([]apiv1.AcornImageBuild, error) {
	builders := &apiv1.AcornImageBuildList{}
	err := c.Client.List(ctx, builders, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	return builders.Items, err
}

func BuildClientID(image, file string) string {
	hashSource := file
	if hashSource == "" {
		hashSource = image
	}
	cwd, _ := os.Getwd()
	id, _ := machineid.ProtectedID("acorn")
	if id == "" {
		id, _ = os.Hostname()
	}
	hashSource = filepath.Join(cwd, hashSource)
	return digest.SHA256(id, hashSource)[:12]
}

func (c *DefaultClient) AcornImageBuild(ctx context.Context, file string, opts *AcornImageBuildOptions) (*v1.AppImage, error) {
	opts, err := opts.complete()
	if err != nil {
		return nil, err
	}

	fileData, err := aml.ReadFile(file)
	if err != nil {
		return nil, err
	}

	vcs := vcs.VCS(file, opts.Cwd)

	builder, err := c.getOrCreateBuilder(ctx, opts.BuilderName)
	if err != nil {
		return nil, err
	}
	opts.BuilderName = builder.Name

	build := &apiv1.AcornImageBuild{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "bld-",
			Namespace:    c.Namespace,
		},
		Spec: v1.AcornImageBuildInstanceSpec{
			ContextCacheKey: BuildClientID("", file),
			BuilderName:     opts.BuilderName,
			Acornfile:       string(fileData),
			Platforms:       opts.Platforms,
			Args:            v1.NewGenericMap(opts.Args),
			Profiles:        opts.Profiles,
			VCS:             vcs,
		},
	}

	err = c.Client.Create(ctx, build)
	if err != nil {
		return nil, err
	}

	dialer := buildclient.WebSocketDialer(websocket.DefaultDialer.DialContext)
	if build.Status.BuildURL == "" {
		dialer = c.Dialer.DialWebsocket
		build.Status.BuildURL = c.RESTClient.Get().
			Namespace(builder.Namespace).
			Resource("builders").
			Name(builder.Name).
			SubResource("port").URL().String()
	}

	if overrideBuildURL := os.Getenv("ACORN_DEBUG_BUILD_URL"); overrideBuildURL != "" {
		logrus.Infof("Overriding build URL [%s] with [%s]", build.Status.BuildURL, overrideBuildURL)
		build.Status.BuildURL = overrideBuildURL
	}

	logrus.Debugf("Building with URL: %s", build.Status.BuildURL)
	return buildclient.Stream(ctx, opts.Cwd, opts.Streams, dialer, (buildclient.CredentialLookup)(opts.Credentials), build)
}
