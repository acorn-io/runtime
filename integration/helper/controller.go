package helper

import (
	"sync"
	"testing"

	"github.com/ibuildthecloud/baaah/pkg/crds"
	"github.com/ibuildthecloud/herd/pkg/controller"
	"github.com/ibuildthecloud/herd/pkg/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	controllerStarted   = false
	controllerStartLock sync.Mutex
)

func EnsureCRDs(t *testing.T) {
	ctx := GetCTX(t)
	if err := crds.Create(ctx, scheme.Scheme, v1.SchemeGroupVersion); err != nil {
		t.Fatal(err)
	}
}

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
		Images: controller.Images{
			AppImageInitImage: images.Images["app-image-init"].Image,
			BuildkitImage:     images.Images["buildkitd"].Image,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Start(GetCTX(t)); err != nil {
		t.Fatal(err)
	}

	controllerStarted = true
}
