package log

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ibuildthecloud/herd/integration/helper"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	hclient "github.com/ibuildthecloud/herd/pkg/client"
	applabels "github.com/ibuildthecloud/herd/pkg/labels"
	"github.com/ibuildthecloud/herd/pkg/log"
	"github.com/ibuildthecloud/herd/pkg/streams"
	"github.com/ibuildthecloud/herd/pkg/watcher"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const sampleLog = `testlog-pod1/cont1-1 line 1-1
testlog-pod1/cont1-1 line 1-2
testlog-pod1/cont2-1 line 1-3
testlog-pod1/cont2-1 line 1-4
testlog-pod2/cont1-2 line 2-1
testlog-pod2/cont1-2 line 2-2
testlog-pod2/cont2-2 line 2-3
testlog-pod2/cont2-2 line 2-4`

func TestLog(t *testing.T) {
	helper.EnsureCRDs(t)
	ctx := helper.GetCTX(t)
	c := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, c)
	app, pod1, pod2 := appPodPod(ns.Name)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	helper.Must(c.Create(ctx, app))

	eg := errgroup.Group{}
	eg.Go(func() error {
		return log.App(ctx, app, &log.Options{
			Client: c,
			Output: streams.Output{
				Out: out,
				Err: errOut,
			},
		})
	})
	podWatcher := watcher.New[*corev1.Pod](c)

	app.Status.Namespace = app.Namespace
	helper.Must(c.Status().Update(ctx, app))
	helper.Must(c.Create(ctx, pod1))
	helper.Must(c.Create(ctx, pod2))
	podWatcher.ByObject(ctx, pod1, func(obj *corev1.Pod) (bool, error) {
		return obj.Status.Phase == corev1.PodSucceeded, nil
	})
	podWatcher.ByObject(ctx, pod2, func(obj *corev1.Pod) (bool, error) {
		return obj.Status.Phase == corev1.PodSucceeded, nil
	})
	var lines []string
	for i := 0; i < 10; i++ {
		lines = strings.Split(strings.TrimSpace(out.String()), "\n")
		if len(lines) == 8 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	helper.Must(c.Delete(ctx, app))
	helper.Must(eg.Wait())
	lines = strings.Split(strings.TrimSpace(out.String()), "\n")
	fmt.Println(out.String())
	assert.Equal(t, 8, len(lines))
	sort.Strings(lines)
	assert.Equal(t, sampleLog, strings.Join(lines, "\n"))
}

func appPodPod(ns string) (*v1.AppInstance, *corev1.Pod, *corev1.Pod) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    ns,
			GenerateName: "herd-test-app-",
		},
		Spec: v1.AppInstanceSpec{},
	}
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testlog-pod1",
			Namespace: ns,
			Labels: map[string]string{
				applabels.HerdAppPod: "true",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			InitContainers: []corev1.Container{
				{
					Name:            "cont1-1",
					Image:           "busybox",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"echo", "-e", "line 1-1\nline 1-2"},
				},
			},
			Containers: []corev1.Container{
				{
					Name:            "cont2-1",
					Image:           "busybox",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"echo", "-e", "line 1-3\nline 1-4"},
				},
			},
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testlog-pod2",
			Namespace: ns,
			Labels: map[string]string{
				applabels.HerdAppPod: "true",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			InitContainers: []corev1.Container{
				{
					Name:            "cont1-2",
					Image:           "busybox",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"echo", "-e", "line 2-1\nline 2-2"},
				},
			},
			Containers: []corev1.Container{
				{
					Name:            "cont2-2",
					Image:           "busybox",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"echo", "-e", "line 2-3\nline 2-4"},
				},
			},
		},
	}

	return app, pod1, pod2
}
