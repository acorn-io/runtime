package log

import (
	"context"
	"fmt"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
)

var (
	colors = []pterm.Color{
		pterm.FgRed,
		pterm.FgGreen,
		pterm.FgYellow,
		pterm.FgBlue,
		pterm.FgMagenta,
		pterm.FgCyan,
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
		if msg.Error == "" {
			key := fmt.Sprintf("%s/%s", msg.PodName, msg.ContainerName)
			color, ok := containerColors[key]
			if !ok {
				color = nextColor()
				containerColors[key] = color
			}

			pterm.Printf("%s: %s\n", color.Sprint(key), msg.Line)
		} else {
			logrus.Error(msg.Error)
		}
	}

	return nil
}
