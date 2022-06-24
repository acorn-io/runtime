package ui

import (
	"context"
	"fmt"

	"github.com/acorn-io/acorn/pkg/ui/server"
	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
)

type Options struct {
	Address     string
	OpenBrowser bool
}

func (o *Options) complete() *Options {
	if o == nil {
		o := Options{}
		return o.complete()
	}
	return o
}

func UI(ctx context.Context, opts *Options) error {
	opts = opts.complete()

	addr, err := server.New(ctx, opts.Address)
	if err != nil {
		return err
	}

	logrus.Infof("Listening on %s", addr)
	if opts.OpenBrowser {
		if err := browser.OpenURL(fmt.Sprintf("http://%s", addr)); err != nil {
			logrus.Errorf("failed to open browser: %s", err)
		}
	}

	<-ctx.Done()
	return nil
}
