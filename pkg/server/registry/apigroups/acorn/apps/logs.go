package apps

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/acorn-io/mink/pkg/strategy"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/k8schannel"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/log"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	clientgo "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewLogs(c client.WithWatch, cfg *clientgo.Config) (*Logs, error) {
	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Logs{
		k8s:    k8s,
		client: c,
	}, nil
}

type Logs struct {
	*strategy.DestroyAdapter
	k8s    kubernetes.Interface
	client client.WithWatch
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
	ns, _ := request.NamespaceFrom(ctx)
	app := &apiv1.App{}
	err := i.client.Get(ctx, kclient.ObjectKey{Namespace: ns, Name: id}, app)
	if err != nil {
		return nil, err
	}

	var (
		opts = options.(*apiv1.LogOptions)
	)

	output := make(chan log.Message)
	go func() {
		defer close(output)
		err := log.App(ctx, app, output, &log.Options{
			Client:           i.client,
			PodClient:        i.k8s.CoreV1(),
			Tail:             opts.Tail,
			Follow:           opts.Follow,
			ContainerReplica: opts.ContainerReplica,
			Container:        opts.Container,
		})
		if err != nil {
			output <- log.Message{
				Err: err,
			}
		}
	}()

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		conn, err := k8schannel.Upgrader.Upgrade(rw, req, nil)
		if err != nil {
			logrus.Errorf("Error during handshake for app logs: %v", err)
			return
		}
		defer conn.Close()

		k8schannel.AddCloseHandler(conn)

		for message := range output {
			lm := apiv1.LogMessage{
				Line:          message.Line,
				ContainerName: message.ContainerName,
				Time:          metav1.NewTime(message.Time),
			}

			if message.Pod != nil {
				lm.AppName = message.Pod.Labels[labels.AcornAppName]
				lm.ContainerName = message.Pod.Name
				if message.ContainerName != message.Pod.Labels[labels.AcornContainerName] {
					lm.ContainerName += "." + message.ContainerName
				}
			}

			if message.Err != nil {
				lm.Error = message.Err.Error()
			}

			data, err := json.Marshal(lm)
			if err != nil {
				panic("failed to marshal update: " + err.Error())
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logrus.Errorf("Error writing log message: %v", err)
				break
			}
		}

		_ = conn.CloseHandler()(websocket.CloseNormalClosure, "")
	}), nil
}

func (i *Logs) ConnectMethods() []string {
	return []string{"GET"}
}
