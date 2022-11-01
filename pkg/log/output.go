package log

import (
	"context"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
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

func Output(ctx context.Context, c client.Client, name string, opts *client.LogOptions) error {
	msgs, err := c.AppLog(ctx, name, opts)
	if err != nil {
		return err
	}

	containerColors := map[string]pterm.Color{}

	for msg := range msgs {
		result, err := SinceLogCheck(opts.Since, msg)
		if err != nil {
			return err
		}
		if result {
			if msg.Error == "" {
				color, ok := containerColors[msg.ContainerName]
				if !ok {
					color = nextColor()
					containerColors[msg.ContainerName] = color
				}

				pterm.Printf("%s: %s\n", color.Sprint(msg.ContainerName), msg.Line)
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
