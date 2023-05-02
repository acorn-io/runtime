package log

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const sampleLog = `line 1-1
line 1-2
line 1-3
line 1-4
line 2-1
line 2-2
line 2-3
line 2-4`

func TestLog(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(context.Background(), image.ID, &client.AppRunOptions{
		/* When running this test with the acorn-linkerd-plugin installed, the app inits too quickly, and
		   the acorn controller does not have enough time to propagate the injection annotation to the
		   test namespace before the app is created, so linkerd does not end up injecting the sidecar.
		   For this reason, we add the annotation here so that it will be placed directly on the app's
		   pods, and the linkerd sidecars will be injected. In clusters without linkerd installed,
		   this annotation will have zero effect. */
		Annotations: []v1.ScopedLabel{
			{
				Key:   "linkerd.io/inject",
				Value: "enabled",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.ContainerStatus["cont1-1"].Ready == 1 &&
			app.Status.ContainerStatus["cont1-2"].Ready == 1
	})

	output, err := c.AppLog(ctx, app.Name, nil)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for msg := range output {
		if msg.Error != "" {
			if len(lines) < 8 && !strings.Contains(msg.Error, "context canceled") {
				t.Fatal(msg.Error)
			}
			continue
		}
		lines = append(lines, msg.Line)
		if len(lines) >= 8 {
			cancel()
		}
	}

	sort.Strings(lines)
	assert.Equal(t, sampleLog, strings.Join(lines, "\n"))
}

func TestContainerLog(t *testing.T) {
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(context.Background(), image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.ContainerStatus["cont1-1"].Ready == 1 &&
			app.Status.ContainerStatus["cont1-2"].Ready == 1
	})

	replicas, err := c.ContainerReplicaList(ctx, &client.ContainerReplicaListOptions{
		App: app.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(replicas, func(i, j int) bool {
		return replicas[i].Name < replicas[j].Name
	})

	output, err := c.AppLog(ctx, app.Name, &client.LogOptions{
		ContainerReplica: replicas[0].Name,
	})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for msg := range output {
		if msg.Error != "" {
			if len(lines) < 8 && !strings.Contains(msg.Error, "context canceled") {
				t.Fatal(msg.Error)
			}
			continue
		}
		lines = append(lines, msg.Line)
		if len(lines) >= 8 {
			cancel()
		}
	}

	sort.Strings(lines)
	assert.Equal(t, "line 1-1\nline 1-2", strings.Join(lines, "\n"))
}

func TestSidecarContainerLog(t *testing.T) {
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(context.Background(), image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.ContainerStatus["cont1-1"].Ready == 1 &&
			app.Status.ContainerStatus["cont1-2"].Ready == 1
	})

	replicas, err := c.ContainerReplicaList(ctx, &client.ContainerReplicaListOptions{
		App: app.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(replicas, func(i, j int) bool {
		return replicas[i].Name < replicas[j].Name
	})

	output, err := c.AppLog(ctx, app.Name, &client.LogOptions{
		ContainerReplica: replicas[1].Name,
	})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for msg := range output {
		if msg.Error != "" {
			if len(lines) < 8 && !strings.Contains(msg.Error, "context canceled") {
				t.Fatal(msg.Error)
			}
			continue
		}
		lines = append(lines, msg.Line)
		if len(lines) >= 8 {
			cancel()
		}
	}

	sort.Strings(lines)
	assert.Equal(t, "line 1-3\nline 1-4", strings.Join(lines, "\n"))
	assert.Len(t, strings.Split(replicas[1].Name, ":"), 2)
}
