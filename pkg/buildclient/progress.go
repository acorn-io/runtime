package buildclient

import (
	"context"

	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/containerd/console"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/sirupsen/logrus"
)

type clientProgressStatus struct {
	streams        *streams.Output
	currentSession string
	progressChan   chan *buildkit.SolveStatus
	doneChan       chan struct{}
	ctx            context.Context
}

func newClientProgress(ctx context.Context, stream *streams.Output) *clientProgressStatus {
	return &clientProgressStatus{
		streams: stream,
		ctx:     ctx,
	}
}

func (c *clientProgressStatus) Display(msg *Message) {
	if msg.StatusSessionID == "" {
		return
	}
	if c.currentSession != msg.StatusSessionID {
		logrus.Debugf("Switching from status session %s => %s", c.currentSession, msg.StatusSessionID)
		c.Close()
		c.progressChan = make(chan *buildkit.SolveStatus, 1)
		c.doneChan = make(chan struct{})
		c.currentSession = msg.StatusSessionID
		go c.display(c.progressChan)
	}
	c.progressChan <- msg.Status
}

func (c *clientProgressStatus) Close() {
	if c.progressChan != nil {
		close(c.progressChan)
		c.progressChan = nil
		<-c.doneChan
	}
}

func (c *clientProgressStatus) display(ch chan *buildkit.SolveStatus) {
	var (
		con console.Console
		err error
	)

	if f, ok := c.streams.Out.(console.File); ok {
		con, err = console.ConsoleFromFile(f)
		if err != nil {
			con = nil
		}
	}

	_, _ = progressui.DisplaySolveStatus(c.ctx, "", con, c.streams.Err, ch)
	close(c.doneChan)
}
