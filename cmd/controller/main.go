package main

import (
	"github.com/ibuildthecloud/herd/integration/helper"
	"github.com/ibuildthecloud/herd/pkg/controller"
	"github.com/ibuildthecloud/herd/pkg/system"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	ctx := signals.SetupSignalHandler()
	if system.AppInitImage == "" {
		images, err := helper.HerdImages(ctx)
		if err != nil {
			logrus.Fatal(err)
		}
		system.AppInitImage = images.Images["app-image-init"].Image
	}
	c, err := controller.New(controller.Config{
		Images: controller.Images{
			AppImageInitImage: system.AppInitImage,
		},
	})
	if err != nil {
		logrus.Fatal(err)
	}
	if err := c.Start(ctx); err != nil {
		logrus.Fatal(err)
	}
}
