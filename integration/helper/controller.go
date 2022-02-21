package helper

import (
	"sync"
	"testing"
	"time"

	"github.com/ibuildthecloud/baaah/pkg/crds"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	hclient "github.com/ibuildthecloud/herd/pkg/client"
	"github.com/ibuildthecloud/herd/pkg/controller"
	"github.com/ibuildthecloud/herd/pkg/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	controllerStarted   = false
	controllerStartLock sync.Mutex
)

func EnsureCRDs(t *testing.T) {
	ctx := GetCTX(t)
	if err := crds.Create(ctx, scheme.Scheme, metav1.SchemeGroupVersion); err != nil {
		t.Fatal(err)
	}
	c, err := hclient.Default()
	if err != nil {
		t.Fatal(err)
	}

	var apps v1.AppInstanceList
	for {
		if err := c.List(ctx, &apps); err != nil {
			time.Sleep(time.Second)
		} else {
			break
		}
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
