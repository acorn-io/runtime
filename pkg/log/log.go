package log

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ibuildthecloud/baaah/pkg/restconfig"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	hclient "github.com/ibuildthecloud/herd/pkg/client"
	applabels "github.com/ibuildthecloud/herd/pkg/labels"
	"github.com/ibuildthecloud/herd/pkg/streams"
	"github.com/ibuildthecloud/herd/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type watchKey struct {
	kind      string
	namespace string
	name      string
}

type watching struct {
	sync.Mutex
	m map[watchKey]bool
}

func (w *watching) shouldWatch(kind, namespace, name string) bool {
	w.Lock()
	defer w.Unlock()
	if w.m == nil {
		w.m = map[watchKey]bool{}
	}

	key := watchKey{
		kind:      kind,
		namespace: namespace,
		name:      name,
	}
	if w.m[key] {
		return false
	}
	w.m[key] = true
	return true
}

type Options struct {
	Output     streams.Output
	RestConfig *rest.Config
	Client     client.WithWatch
	PodClient  v12.PodsGetter
	TailLines  *int64
	Timestamps bool

	outputLocked bool
}

func (o *Options) restConfig() (*rest.Config, error) {
	if o.RestConfig != nil {
		return o.RestConfig, nil
	}
	if o.Client == nil {
		return nil, fmt.Errorf("RestConfig or Client field must be set")
	}
	var err error
	o.RestConfig, err = restconfig.New(o.Client.Scheme())
	return o.RestConfig, err
}

func (o *Options) Complete() (*Options, error) {
	if o == nil {
		o = &Options{}
	}

	if !o.outputLocked {
		o.Output = o.Output.Locked()
	}

	if o.PodClient == nil {
		cfg, err := o.restConfig()
		if err != nil {
			return o, err
		}
		cs, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return o, err
		}
		o.PodClient = cs.CoreV1()
	}

	if o.Client == nil {
		cfg, err := o.restConfig()
		if err != nil {
			return o, err
		}
		c, err := hclient.New(cfg)
		if err != nil {
			return o, err
		}
		o.Client = c
	}

	return o, nil
}

func pipe(input io.ReadCloser, output streams.Output, prefix string, timestamps bool, after *metav1.Time) (*metav1.Time, error) {
	defer input.Close()

	var lastTS *metav1.Time

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		ts, line, _ := strings.Cut(line, " ")

		pt, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return lastTS, err
		}
		lastTS = &metav1.Time{
			Time: pt.Local(),
		}
		if after != nil && !lastTS.After(after.Time) {
			continue
		}

		lineBuffer := &bytes.Buffer{}
		if timestamps {
			lineBuffer.WriteString(line)
			lineBuffer.WriteString(" ")
		}
		if prefix != "" {
			lineBuffer.WriteString(prefix)
			lineBuffer.WriteString(" ")
		}
		lineBuffer.WriteString(line)
		lineBuffer.WriteString("\n")

		_, err = output.Out.Write(lineBuffer.Bytes())
		if err != nil {
			return lastTS, err
		}
	}

	return lastTS, scanner.Err()
}

func Container(ctx context.Context, pod *corev1.Pod, name string, options *Options) (err error) {
	options, err = options.Complete()
	if err != nil {
		return err
	}

	var (
		first     = true
		since     *metav1.Time
		tailLines = options.TailLines
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if first {
			first = false
		} else {
			time.Sleep(time.Second)
		}

		req := options.PodClient.Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container:  name,
			Follow:     true,
			SinceTime:  since,
			Timestamps: true,
			TailLines:  tailLines,
		})
		readCloser, err := req.Stream(ctx)
		if err != nil {
			logrus.Debugf("failed to get logs for container %s on pod %s/%s: %v", name, pod.Namespace, pod.Name, err)
			continue
		}
		// pipe will close the readCloser
		lastTS, err := pipe(readCloser, options.Output, pod.Name+"/"+name, options.Timestamps, since)
		if err != nil {
			logrus.Debugf("failed to stream logs for container %s on pod %s/%s: %v", name, pod.Namespace, pod.Name, err)
		}
		if lastTS != nil {
			since = lastTS
			tailLines = nil
		}
	}
}

func Pod(ctx context.Context, pod *corev1.Pod, options *Options) error {
	options, err := options.Complete()
	if err != nil {
		return err
	}

	podWatcher := watcher.New[*corev1.Pod](options.Client)
	watching := watching{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err = podWatcher.ByName(ctx, pod.Namespace, pod.Name, func(pod *corev1.Pod) (bool, error) {
		if !pod.DeletionTimestamp.IsZero() {
			return true, nil
		}
		for _, container := range pod.Spec.Containers {
			if watching.shouldWatch("container", pod.Namespace, container.Name) {
				go Container(ctx, pod, container.Name, options)
			}
		}
		for _, container := range pod.Spec.InitContainers {
			if watching.shouldWatch("initcontainer", pod.Namespace, container.Name) {
				go Container(ctx, pod, container.Name, options)
			}
		}
		return false, nil
	})
	return err
}

func App(ctx context.Context, app *v1.AppInstance, options *Options) error {
	options, err := options.Complete()
	if err != nil {
		return err
	}

	var (
		appWatcher = watcher.New[*v1.AppInstance](options.Client)
		podWatcher = watcher.New[*corev1.Pod](options.Client)
		watching   = watching{}
	)

	app, err = appWatcher.ByName(ctx, app.Namespace, app.Name, func(app *v1.AppInstance) (bool, error) {
		return app.Status.Namespace != "", nil
	})
	if err != nil {
		return err
	}

	podSelector := labels.SelectorFromSet(labels.Set{
		applabels.HerdAppPod: "true",
	})

	// Ensure that if once func finishes they are all canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg := errgroup.Group{}
	eg.Go(func() error {
		// Don't recursively watch if app's pods are in the same namespace.
		if app.Status.Namespace == app.Namespace {
			return nil
		}
		_, err := appWatcher.BySelector(ctx, app.Status.Namespace, labels.Everything(), func(app *v1.AppInstance) (bool, error) {
			if watching.shouldWatch("AppInstance", app.Namespace, app.Name) {
				go App(ctx, app, options)
			}
			return false, nil
		})
		return err
	})
	eg.Go(func() error {
		_, err := podWatcher.BySelector(ctx, app.Status.Namespace, podSelector, func(pod *corev1.Pod) (bool, error) {
			if watching.shouldWatch("Pod", pod.Namespace, pod.Name) {
				go Pod(ctx, pod, options)
			}
			return false, nil
		})
		return err
	})
	eg.Go(func() error {
		defer cancel()
		_, err := appWatcher.ByObject(ctx, app, func(app *v1.AppInstance) (bool, error) {
			return !app.DeletionTimestamp.IsZero(), nil
		})
		return err
	})

	err = eg.Wait()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
