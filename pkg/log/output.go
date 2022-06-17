package log

import (
	"context"
	"fmt"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/sirupsen/logrus"
)

func Output(ctx context.Context, c client.Client, name string, opts *client.LogOptions) error {
	msgs, err := c.AppLog(ctx, name, opts)
	if err != nil {
		return err
	}

	for msg := range msgs {
		if msg.Error == "" {
			fmt.Printf("%s/%s: %s\n", msg.PodName, msg.ContainerName, msg.Line)
		} else {
			logrus.Error(msg.Error)
		}
	}

	return nil
}
