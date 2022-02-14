package helper

import (
	"sync"
	"testing"

	"github.com/ibuildthecloud/herd/pkg/controller"
)

var (
	controllerStarted   = false
	controllerStartLock sync.Mutex
)

func StartController(t *testing.T) {
	controllerStartLock.Lock()
	defer controllerStartLock.Unlock()

	if controllerStarted {
		return
	}

	images, err := HerdImages(GetCTX(t))
	if err != nil {
		t.Fatal(err)
	}

	c, err := controller.New(controller.Config{
		AppImageInitImage: images.Images["app-image-init"].Image,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Start(GetCTX(t)); err != nil {
		t.Fatal(err)
	}

	controllerStarted = true
}
