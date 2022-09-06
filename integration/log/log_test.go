package log

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	applabels "github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
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
	logrus.SetLevel(logrus.DebugLevel)
	ti := time.Now()
	helper.EnsureCRDs(t)
	ctx, cancel := context.WithTimeout(helper.GetCTX(t), time.Minute)
	defer cancel()

	c := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, c)
	app, pod1, pod2 := appPodPod(ns.Name)
	helper.Must(c.Create(ctx, app))
	fmt.Printf("After app create %v !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n", time.Since(ti))
	for {
		app.Status.Namespace = app.Namespace
		err := c.Status().Update(ctx, app)
		if apierror.IsConflict(err) {
			err := c.Get(ctx, router.Key(app.Namespace, app.Name), app)
			if err != nil {
				t.Fatal(err)
			}
			continue
		}
		break
	}
	helper.Must(c.Create(ctx, pod1))
	fmt.Printf("After pod1 create %v !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n", time.Since(ti))
	helper.Must(c.Create(ctx, pod2))
	fmt.Printf("After pod2 create %v !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n", time.Since(ti))

	output := make(chan log.Message)
	go func() {
		e := log.App(ctx, app, output, &log.Options{
			Client: c,
			Follow: true,
		})
		fmt.Printf("GOT E: %v @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n", e)
		fmt.Printf("After got-e create %v !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n", time.Since(ti))
		close(output)
	}()

	var lines []string
	for msg := range output {
		if msg.Err != nil {
			fmt.Printf("GOT msg.err: %v @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n", msg.Err)
			fmt.Printf("After msg.err create %v !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n", time.Since(ti))
			if len(lines) < 8 && !strings.Contains(msg.Err.Error(), "context canceled") {
				t.Fatal(msg.Err)
			}
			continue
		}
		fmt.Println("LINE: ", fmt.Sprintf("%s/%s %s", msg.Pod.Name, msg.ContainerName, msg.Line))
		lines = append(lines, fmt.Sprintf("%s/%s %s", msg.Pod.Name, msg.ContainerName, msg.Line))
		if len(lines) >= 8 {
			cancel()
		}
	}

	sort.Strings(lines)
	assert.Equal(t, sampleLog, strings.Join(lines, "\n"))
}

func appPodPod(ns string) (*v1.AppInstance, *corev1.Pod, *corev1.Pod) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    ns,
			GenerateName: "acorn-test-app-",
		},
		Spec: v1.AppInstanceSpec{},
	}
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testlog-pod1",
			Namespace: ns,
			Labels: map[string]string{
				applabels.AcornManaged: "true",
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
				applabels.AcornManaged: "true",
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
