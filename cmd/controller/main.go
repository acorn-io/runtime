package main

import (
	"github.com/ibuildthecloud/herd/pkg/controller"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	ctx := signals.SetupSignalHandler()
	c, err := controller.New()
	if err != nil {
		logrus.Fatal(err)
	}
	if err := c.Start(ctx); err != nil {
		logrus.Fatal(err)
	}
	<-ctx.Done()
}
