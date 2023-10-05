package log

import (
	"context"
	"strings"
	"time"

	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
)

var (
	colors = []pterm.Color{
		pterm.FgGreen,
		pterm.FgYellow,
		pterm.FgBlue,
		pterm.FgMagenta,
		pterm.FgCyan,
		pterm.FgRed,
	}
	index = 0
)

func nextColor() pterm.Color {
	c := colors[index%len(colors)]
	index++
	return c
}

func getLogger(opts *client.LogOptions) client.ContainerLogsWriter {
	if opts.Logger == nil {
		return &DefaultLoggerImpl{
			containerColors: map[string]pterm.Color{},
		}
	}
	return opts.Logger
}

func Output(ctx context.Context, c client.Client, name string, opts *client.LogOptions) error {
	msgs, err := c.AppLog(ctx, name, opts)
	if err != nil {
		return err
	}

	logger := getLogger(opts)

	for msg := range msgs {
		result, err := SinceLogCheck(opts.Since, msg)
		if err != nil {
			return err
		}
		if result {
			if msg.Error == "" {
				logger.Container(msg.Time, msg.ContainerName, msg.Line)
			} else if !strings.Contains(msg.Error, "context canceled") {
				logrus.Error(msg.Error)
			}
		}
	}

	return nil
}

func SinceLogCheck(since string, msg v1.LogMessage) (bool, error) {
	if since == "" {
		return true, nil
	}
	beforeTime, err := time.ParseDuration(since)
	if err != nil {
		return false, err
	}
	return msg.Time.After(time.Now().Add(-beforeTime)), nil
}
