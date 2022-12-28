package buildclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/containerd/console"
	"github.com/gorilla/websocket"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/pkg/errors"
)

func wsURL(url string) string {
	if strings.HasPrefix(url, "http") {
		return strings.Replace(url, "http", "ws", 1)
	}
	return url
}

type WebSocketDialer func(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)

func Stream(ctx context.Context, cwd string, streams *streams.Output, dialer WebSocketDialer,
	build *apiv1.AcornImageBuild) (*v1.AppImage, error) {
	conn, _, err := dialer(ctx, wsURL(build.Status.BuildURL), map[string][]string{
		"X-Acorn-Build-Token": {build.Status.Token},
	})
	if err != nil {
		return nil, err
	}

	var (
		messages = NewWebsocketMessages(conn)
		syncers  = map[string]*fileSyncClient{}
	)
	defer func() {
		for _, s := range syncers {
			s.Close()
		}
	}()
	defer messages.Close()

	msgs, cancel := messages.Recv()
	defer cancel()

	var progressChan *chan *buildkit.SolveStatus
	if streams != nil {
		c, done := clientProgress(ctx, streams)
		defer func() { close(c); <-done }()
		progressChan = &c
	}

	// Handle messages synchronous since new subscribers are started
	// and we don't want to miss a message.
	messages.OnMessage(func(msg *Message) error {
		if msg.FileSessionID == "" {
			return nil
		}
		if _, ok := syncers[msg.FileSessionID]; ok {
			return nil
		}
		s, err := newFileSyncClient(ctx, cwd, msg.FileSessionID, messages, msg.SyncOptions)
		if err != nil {
			return err
		}
		syncers[msg.FileSessionID] = s
		return nil
	})

	messages.Start(ctx)

	for msg := range msgs {
		if msg.Status != nil && progressChan != nil {
			*progressChan <- msg.Status
		} else if msg.AppImage != nil {
			return msg.AppImage, nil
		} else if msg.Error != "" {
			return nil, errors.New(msg.Error)
		}
	}

	return nil, fmt.Errorf("build failed")
}

func clientProgress(ctx context.Context, streams *streams.Output) (chan *buildkit.SolveStatus, chan struct{}) {
	var (
		c    console.Console
		err  error
		done = make(chan struct{})
	)

	if f, ok := streams.Out.(console.File); ok {
		c, err = console.ConsoleFromFile(f)
		if err != nil {
			c = nil
		}
	}

	ch := make(chan *buildkit.SolveStatus, 1)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		_, _ = progressui.DisplaySolveStatus(ctx, "", c, streams.Err, ch)
		close(done)
	}()
	return ch, done
}
