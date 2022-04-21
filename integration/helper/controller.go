package helper

import (
	"context"
	"sync"
	"testing"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/crds"
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

	c, err := controller.New()
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	controllerStarted = true
}
