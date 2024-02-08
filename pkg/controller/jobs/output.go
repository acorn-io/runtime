package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/urlbuilder"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client/term"
	"github.com/acorn-io/runtime/pkg/jobs"
	"github.com/acorn-io/runtime/pkg/k8schannel"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/streams"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getAppGeneration(ctx context.Context, c kclient.Client, namespace, jobName string) (string, error) {
	job := batchv1.Job{}
	if err := c.Get(ctx, router.Key(namespace, jobName), &job); apierrors.IsNotFound(err) {
		cronJob := batchv1.CronJob{}
		if err := c.Get(ctx, router.Key(namespace, jobName), &cronJob); err != nil {
			return "", err
		}
		return cronJob.Annotations[labels.AcornAppGeneration], nil
	} else if err != nil {
		return "", err
	}
	return job.Annotations[labels.AcornAppGeneration], nil
}

type Handler struct {
	Dialer *k8schannel.Dialer
	Server *url.URL
}

func NewHandler(cfg *rest.Config) (*Handler, error) {
	d, err := k8schannel.NewDialer(cfg, false)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}

	return &Handler{
		Dialer: d,
		Server: u,
	}, nil
}

func (h *Handler) runCommand(ctx context.Context, pod *corev1.Pod, command ...string) (map[string][]byte, error) {
	execURL := urlbuilder.PathBuilder{
		Prefix:      "/api",
		APIGroup:    "",
		APIVersion:  "v1",
		Namespace:   pod.Namespace,
		Name:        pod.Name,
		Resource:    "pods",
		Subresource: "exec",
	}.URL(h.Server)
	execURL.RawQuery = url.Values{
		"stdout":    []string{"true"},
		"stderr":    []string{"true"},
		"container": []string{jobs.Helper},
		"command":   command,
	}.Encode()
	conn, err := h.Dialer.DialContext(ctx, execURL.String(), nil)
	if err != nil {
		return nil, err
	}

	output := &streams.Streams{
		Output: streams.Output{
			Out: &bytes.Buffer{},
			Err: &bytes.Buffer{},
		},
		In: &bytes.Buffer{},
	}

	io := conn.ToExecIO(false)
	code, err := term.Pipe(io, output)
	if err != nil {
		return nil, err
	}

	if code != 0 {
		buf := bytes.NewBuffer(output.Out.(*bytes.Buffer).Bytes())
		errBuf := bytes.NewBuffer(output.Err.(*bytes.Buffer).Bytes())
		if buf.Len() > 0 && errBuf.Len() > 0 {
			buf.WriteString("; ")
		}
		buf.Write(errBuf.Bytes())
		return nil, fmt.Errorf("exit code %d: %s", code, buf)
	}

	return map[string][]byte{
		"out": output.Out.(*bytes.Buffer).Bytes(),
		"err": output.Err.(*bytes.Buffer).Bytes(),
	}, nil
}

func (h *Handler) SaveJobOutput(req router.Request, _ router.Response) error {
	pod := req.Object.(*corev1.Pod)
	jobName := pod.Labels[labels.AcornJobName]

	if jobName == "" || pod.Status.Phase != corev1.PodRunning {
		return nil
	}

	spec := pod.Annotations[labels.AcornContainerSpec]
	if spec == "" {
		return nil
	}

	container := v1.Container{}
	if err := json.Unmarshal([]byte(spec), &container); err != nil {
		return err
	}

	names := sets.New[string](jobName)
	for name := range container.Sidecars {
		names.Insert(name)
	}

	helperRunning := false
	for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
		if status.Name == jobs.Helper && status.State.Running != nil {
			helperRunning = true
		}
		if status.State.Terminated != nil {
			names.Delete(status.Name)
		}
	}

	if names.Len() > 0 || !helperRunning {
		return nil
	}

	generation, err := getAppGeneration(req.Ctx, req.Client, pod.Namespace, jobName)
	if err != nil {
		return err
	}

	secretName := jobs.GetJobOutputSecretName(pod.Namespace, jobName)

	data, err := h.runCommand(req.Ctx, pod, "/usr/local/bin/acorn-job-get-output")
	if err != nil {
		return err
	}

	err = apply.New(req.Client).Ensure(req.Ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: pod.Namespace,
			Labels: labels.ManagedByApp(pod.Labels[labels.AcornAppNamespace], pod.Labels[labels.AcornAppName],
				labels.AcornAppGeneration, generation),
		},
		Data: data,
	})
	if err != nil {
		return err
	}

	_, err = h.runCommand(req.Ctx, pod, "/usr/local/bin/acorn-job-helper-shutdown")
	return err
}
