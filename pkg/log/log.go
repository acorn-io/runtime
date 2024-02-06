package log

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/watcher"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	hclient "github.com/acorn-io/runtime/pkg/k8sclient"
	applabels "github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/strings/slices"
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

func objectDeleted(obj metav1.Object) bool {
	return !obj.GetDeletionTimestamp().IsZero() && len(obj.GetFinalizers()) == 0
}

type Message struct {
	Line          string
	Pod           *corev1.Pod
	ContainerName string
	Time          time.Time

	Err error
}

type Options struct {
	RestConfig       *rest.Config
	Client           client.WithWatch
	PodClient        v12.PodsGetter
	Tail             *int64
	Follow           bool
	ContainerReplica string
	Container        string
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
	scanner.Buffer(nil, 2_000_000)
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
	options, err = options.Complete()
	if err != nil {
		return err
	}

	var (
		first = true
		since *metav1.Time
		tail  = options.Tail
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
			TailLines:  tail,
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
			if err == nil && (objectDeleted(pod) || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed) {
				return nil
			}
		}
		if lastTS != nil {
			since = lastTS
			tail = nil
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
	options, err := options.Complete()
	if err != nil {
		return err
	}

	if !options.Follow {
		if objectDeleted(pod) {
			return nil
		}
		for _, container := range pod.Spec.Containers {
			if !matchesContainer(pod, container, options) {
				continue
			}
			if err := Container(ctx, pod, container.Name, output, options); err != nil {
				return err
			}
		}
		for _, container := range pod.Spec.InitContainers {
			if !matchesContainer(pod, container, options) {
				continue
			}
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
			if objectDeleted(pod) {
				return true, nil
			}
			for _, container := range pod.Spec.Containers {
				container := container
				if !matchesContainer(pod, container, options) {
					continue
				}
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
				if !matchesContainer(pod, container, options) {
					continue
				}
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

func appNoFollow(ctx context.Context, app *apiv1.App, output chan<- Message, options *Options) error {
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
		if !matchesPod(&pod, options) {
			continue
		}
		if err := Pod(ctx, &pod, output, options); err != nil {
			return err
		}
	}

	apps := &apiv1.AppList{}
	err = options.Client.List(ctx, apps, &client.ListOptions{
		Namespace:     app.Namespace,
		LabelSelector: podSelector,
	})
	if err != nil {
		return err
	}

	for _, child := range apps.Items {
		if strings.HasPrefix(child.Name, app.Name+".") {
			if err := appNoFollow(ctx, &child, output, options); err != nil {
				return err
			}
		}
	}

	return nil
}

func matchesPod(pod *corev1.Pod, options *Options) bool {
	if options == nil || options.ContainerReplica == "" {
		return true
	}
	podName, _ := publicname.SplitPodContainerName(options.ContainerReplica)
	return pod.Name == podName
}

func matchesContainer(pod *corev1.Pod, container corev1.Container, options *Options) bool {
	if options != nil && options.ContainerReplica != "" {
		podName, containerName := publicname.SplitPodContainerName(options.ContainerReplica)
		if containerName == "" {
			if pod.Labels[applabels.AcornContainerName] != "" {
				return pod.Name == podName && container.Name == pod.Labels[applabels.AcornContainerName]
			} else if pod.Labels[applabels.AcornFunctionName] != "" {
				return pod.Name == podName && container.Name == pod.Labels[applabels.AcornFunctionName]
			}
			return pod.Name == podName && container.Name == pod.Labels[applabels.AcornJobName]
		}
		return pod.Name == podName && container.Name == containerName
	}

	// user has selected a specific acorn container name (the name seen in the acornfile containers section)
	if options != nil && options.Container != "" {
		// Must match the acorn container name or job name on the pod
		if pod.Labels[applabels.AcornContainerName] != options.Container &&
			pod.Labels[applabels.AcornFunctionName] != options.Container &&
			pod.Labels[applabels.AcornJobName] != options.Container {
			return false
		}
	}

	var validContainerNames []string
	if pod.Labels[applabels.AcornContainerName] != "" {
		validContainerNames = append(validContainerNames, pod.Labels[applabels.AcornContainerName])
	}
	if pod.Labels[applabels.AcornFunctionName] != "" {
		validContainerNames = append(validContainerNames, pod.Labels[applabels.AcornFunctionName])
	}
	if pod.Labels[applabels.AcornJobName] != "" {
		validContainerNames = append(validContainerNames, pod.Labels[applabels.AcornJobName])
	}

	containerSpecData := []byte(pod.Annotations[applabels.AcornContainerSpec])
	if len(containerSpecData) == 0 {
		return false
	}

	containerSpec := &internalv1.Container{}
	err := json.Unmarshal(containerSpecData, containerSpec)
	if err != nil {
		logrus.Errorf("failed to unmarshal container spec for %s/%s: %s",
			pod.Namespace, pod.Name, containerSpecData)
		return false
	}

	for sidecarName := range containerSpec.Sidecars {
		validContainerNames = append(validContainerNames, sidecarName)
	}

	return slices.Contains(validContainerNames, container.Name)
}

func App(ctx context.Context, app *apiv1.App, output chan<- Message, options *Options) error {
	options, err := options.Complete()
	if err != nil {
		return err
	}

	if !options.Follow {
		return appNoFollow(ctx, app, output, options)
	}

	var (
		appWatcher = watcher.New[*apiv1.App](options.Client)
		podWatcher = watcher.New[*corev1.Pod](options.Client)
		watching   = watching{}
	)

	app, err = appWatcher.ByName(ctx, app.Namespace, app.Name, func(app *apiv1.App) (bool, error) {
		return app.Status.Namespace != "", nil
	})
	if err != nil {
		return err
	}

	podSelector := labels.SelectorFromSet(labels.Set{
		applabels.AcornManaged: "true",
	})
	parentSelector := labels.SelectorFromSet(labels.Set{
		applabels.AcornParentAcornName: app.Name,
	})

	// Ensure that if once func finishes they are all canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg := errgroup.Group{}
	eg.Go(func() error {
		defer cancel()
		_, err := appWatcher.BySelector(ctx, app.Namespace, parentSelector, func(app *apiv1.App) (bool, error) {
			if watching.shouldWatch("App", app.Namespace, app.Name) {
				eg.Go(func() error {
					err := App(ctx, app, output, options)
					if err != nil {
						output <- Message{
							Time: time.Now(),
							Err:  err,
						}
					}
					return err
				})
			}
			return false, nil
		})
		return err
	})
	eg.Go(func() error {
		defer cancel()
		_, err := podWatcher.BySelector(ctx, app.Status.Namespace, podSelector, func(pod *corev1.Pod) (bool, error) {
			if !matchesPod(pod, options) {
				return false, nil
			}
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
					return err
				})
			}
			return false, nil
		})
		return err
	})
	eg.Go(func() error {
		defer cancel()
		_, err := appWatcher.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
			return objectDeleted(app), nil
		})
		return err
	})

	err = eg.Wait()
	if errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}
