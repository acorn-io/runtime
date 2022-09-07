package log

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	applabels "github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/watcher"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

type Message struct {
	Line          string
	Pod           *corev1.Pod
	ContainerName string
	Time          time.Time

	Err error
}

type Options struct {
	RestConfig *rest.Config
	Client     client.WithWatch
	PodClient  v12.PodsGetter
	TailLines  *int64
	Follow     bool
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

func pipe(input io.ReadCloser, output chan<- Message, pod *corev1.Pod, name string, after *metav1.Time) (*metav1.Time, error) {
	defer input.Close()

	var lastTS *metav1.Time

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		ts, newLine, _ := strings.Cut(line, " ")

		pt, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			newLine = line
			pt = time.Time{}
			//return lastTS, err
		}
		lastTS = &metav1.Time{
			Time: pt.Local(),
		}
		if after != nil && !lastTS.After(after.Time) {
			continue
		}

		output <- Message{
			Line:          newLine,
			Pod:           pod,
			ContainerName: name,
			Time:          lastTS.Time,
		}
	}

	return lastTS, scanner.Err()
}

func Container(ctx context.Context, pod *corev1.Pod, name string, output chan<- Message, options *Options) (err error) {
	logrus.Debugf("NOW WERE IN CONTAINER %v", pod)
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
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
			pod, err := options.PodClient.Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			if err == nil && !isContainerLoggable(pod, name) {
				continue
			}
		}

		req := options.PodClient.Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container:  name,
			Follow:     options.Follow,
			SinceTime:  since,
			Timestamps: true,
			TailLines:  tailLines,
		})
		readCloser, err := req.Stream(ctx)
		if err != nil {
			pod, newErr := options.PodClient.Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(newErr) {
				return newErr
			}
			output <- Message{
				Time:          time.Now(),
				Pod:           pod,
				ContainerName: name,
				Err:           fmt.Errorf("failed to get logs for container %s on pod %s/%s: %v", name, pod.Namespace, pod.Name, err),
			}
			continue
		}
		// pipe will close the readCloser
		lastTS, err := pipe(readCloser, output, pod, name, since)
		if err != nil && !errors.Is(err, context.Canceled) {
			output <- Message{
				Time:          time.Now(),
				Pod:           pod,
				ContainerName: name,
				Err:           fmt.Errorf("failed to stream logs for container %s on pod %s/%s: %v", name, pod.Namespace, pod.Name, err),
			}
		} else if err == nil {
			pod, err := options.PodClient.Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			if err == nil && (!pod.DeletionTimestamp.IsZero() || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed) {
				return nil
			}
		}
		if lastTS != nil {
			since = lastTS
			tailLines = nil
		}

		if !options.Follow {
			break
		}
	}

	return nil
}

func isContainerLoggable(pod *corev1.Pod, containerName string) bool {
	for _, status := range pod.Status.InitContainerStatuses {
		if status.Name == containerName &&
			(status.State.Running != nil || status.State.Terminated != nil || status.LastTerminationState.Terminated != nil) {
			return true
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName &&
			(status.State.Running != nil || status.State.Terminated != nil || status.LastTerminationState.Terminated != nil) {
			return true
		}
	}
	return false
}

func Pod(ctx context.Context, pod *corev1.Pod, output chan<- Message, options *Options) error {
	logrus.Debugf("NOW WERE ARE IN POD")
	options, err := options.Complete()
	if err != nil {
		return err
	}

	if !options.Follow {
		if !pod.DeletionTimestamp.IsZero() {
			return nil
		}
		for _, container := range pod.Spec.Containers {
			if err := Container(ctx, pod, container.Name, output, options); err != nil {
				return err
			}
		}
		for _, container := range pod.Spec.InitContainers {
			if err := Container(ctx, pod, container.Name, output, options); err != nil {
				return err
			}
		}
		return nil
	}

	podWatcher := watcher.New[*corev1.Pod](options.Client)
	watching := watching{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer cancel()
		_, err = podWatcher.ByName(ctx, pod.Namespace, pod.Name, func(pod *corev1.Pod) (bool, error) {
			if !pod.DeletionTimestamp.IsZero() {
				return true, nil
			}
			for _, container := range pod.Spec.Containers {
				container := container
				if !isContainerLoggable(pod, container.Name) {
					continue
				}
				if watching.shouldWatch("container", pod.Name, container.Name) {
					eg.Go(func() error {
						err := Container(ctx, pod, container.Name, output, options)
						if err != nil {
							output <- Message{
								Pod:           pod,
								ContainerName: container.Name,
								Time:          time.Now(),
								Err:           err,
							}
						}
						return nil
					})
				}
			}
			for _, container := range pod.Spec.InitContainers {
				container := container
				if !isContainerLoggable(pod, container.Name) {
					continue
				}
				if watching.shouldWatch("initcontainer", pod.Name, container.Name) {
					eg.Go(func() error {
						err := Container(ctx, pod, container.Name, output, options)
						if err != nil {
							output <- Message{
								Pod:           pod,
								ContainerName: container.Name,
								Time:          time.Now(),
								Err:           err,
							}
						}
						return nil
					})
				}
			}
			return false, nil
		})
		return err
	})

	return eg.Wait()
}

func appNoFollow(ctx context.Context, app *v1.AppInstance, output chan<- Message, options *Options) error {
	if app.Status.Namespace == "" {
		return nil
	}

	podSelector := labels.SelectorFromSet(labels.Set{
		applabels.AcornManaged: "true",
	})

	pods := &corev1.PodList{}
	err := options.Client.List(ctx, pods, &client.ListOptions{
		Namespace:     app.Status.Namespace,
		LabelSelector: podSelector,
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		if err := Pod(ctx, &pod, output, options); err != nil {
			return err
		}
	}

	apps := &v1.AppInstanceList{}
	err = options.Client.List(ctx, apps, &client.ListOptions{
		Namespace: app.Status.Namespace,
	})
	if err != nil {
		return err
	}

	for _, app := range apps.Items {
		if err := App(ctx, &app, output, options); err != nil {
			return err
		}
	}

	return nil
}

func App(ctx context.Context, app *v1.AppInstance, output chan<- Message, options *Options) error {
	options, err := options.Complete()
	if err != nil {
		return err
	}

	if !options.Follow {
		return appNoFollow(ctx, app, output, options)
	}

	var (
		appWatcher = watcher.New[*v1.AppInstance](options.Client)
		podWatcher = watcher.New[*corev1.Pod](options.Client)
		watching   = watching{}
	)

	app, err = appWatcher.ByName(ctx, app.Namespace, app.Name, func(app *v1.AppInstance) (bool, error) {
		logrus.Debugf("OK.................. %#v", app)
		return app.Status.Namespace != "", nil
	})
	if err != nil {
		return err
	}

	podSelector := labels.SelectorFromSet(labels.Set{
		applabels.AcornManaged: "true",
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
			logrus.Debugf("got pod eeeeeeeeeeeeveeeeeeeeeent %#v", app)
			if watching.shouldWatch("AppInstance", app.Namespace, app.Name) {
				eg.Go(func() error {
					logrus.Debugf("GOING RECURSIVE BC I HATE CRAIG.............. %#v", app)
					err := App(ctx, app, output, options)
					if err != nil {
						output <- Message{
							Time: time.Now(),
							Err:  err,
						}
					}
					return nil
				})
			} else {
				logrus.Debugf("app nnnnnnooooooooooooooppppppppeeeeee %#v", app)
			}
			return false, nil
		})
		return err
	})
	eg.Go(func() error {
		defer cancel()
		_, err := podWatcher.BySelector(ctx, app.Status.Namespace, podSelector, func(pod *corev1.Pod) (bool, error) {
			logrus.Debugf("got pod eeeeeeeeeeeeveeeeeeeeeent %#v", pod)
			if watching.shouldWatch("Pod", pod.Namespace, pod.Name) {
				eg.Go(func() error {
					err := Pod(ctx, pod, output, options)
					if err != nil {
						output <- Message{
							Pod:  pod,
							Time: time.Now(),
							Err:  err,
						}
					}
					return nil
				})
			} else {
				logrus.Debugf("nnnnnnooooooooooooooppppppppeeeeee %#v", pod)
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
