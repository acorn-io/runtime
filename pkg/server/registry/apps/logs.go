package apps

import (
	"context"
	"encoding/json"
	"net/http"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	clientgo "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewLogs(c client.WithWatch, apps *Storage, cfg *clientgo.Config) (*Logs, error) {
	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Logs{
		k8s:    k8s,
		client: c,
		apps:   apps,
	}, nil
}

type Logs struct {
	k8s    kubernetes.Interface
	client client.WithWatch
	apps   *Storage
}

func (i *Logs) NamespaceScoped() bool {
	return true
}

func (i *Logs) New() runtime.Object {
	return &apiv1.LogOptions{}
}

func (i *Logs) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.LogOptions{}, false, ""
}

func (i *Logs) Connect(ctx context.Context, id string, options runtime.Object, r rest.Responder) (http.Handler, error) {
	obj, err := i.apps.Get(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	var (
		opts = options.(*apiv1.LogOptions)
		app  = obj.(*apiv1.App)
	)

	output := make(chan log.Message)
	go func() {
		defer close(output)
		err := log.App(ctx, &v1.AppInstance{
			ObjectMeta: app.ObjectMeta,
			Spec:       app.Spec,
			Status:     app.Status,
		}, output, &log.Options{
			Client:    i.client,
			PodClient: i.k8s.CoreV1(),
			TailLines: opts.TailLines,
			Follow:    opts.Follow,
		})
		if err != nil {
			output <- log.Message{
				Err: err,
			}
		}
	}()

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		if f, ok := rw.(http.Flusher); ok {
			f.Flush()
		}
		for message := range output {
			lm := apiv1.LogMessage{
				Line:          message.Line,
				ContainerName: message.ContainerName,
				Time:          metav1.NewTime(message.Time),
			}

			if message.Pod != nil {
				lm.PodName = message.Pod.Name
			}

			if message.Err != nil {
				lm.Error = message.Err.Error()
			}

			data, err := json.Marshal(lm)
			if err != nil {
				panic("failed to marshal update: " + err.Error())
			}
			_, _ = rw.Write(append(data, '\n'))
			if f, ok := rw.(http.Flusher); ok {
				f.Flush()
			}
		}
	}), nil
}

func (i *Logs) ConnectMethods() []string {
	return []string{"GET"}
}
