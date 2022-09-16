package image

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	reg "github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
)

var (
	images = []string{
		"test-repo/test:latest",
		"test-repo/test:head",
		"test-repo/test2:latest",
		"test-repo/test2:head",
	}
)

func Registry(t *testing.T) name.Registry {
	t.Helper()

	reg := reg.New()
	srv := httptest.NewServer(reg)

	address := srv.Listener.Addr().String()
	ref, err := name.ParseReference(address + "/test-repo/test:latest")
	if err != nil {
		t.Fatal(err)
	}

	if err := remote.Write(ref, empty.Image); err != nil {
		t.Fatal(err)
	}

	for _, imageName := range images {
		ref, err := name.NewTag(address + "/" + imageName)
		if err != nil {
			t.Fatal(err)
		}
		err = remote.Tag(ref, empty.Image)
		if err != nil {
			t.Fatal(err)
		}
	}

	regRef, err := name.NewRegistry(address)
	if err != nil {
		t.Fatal(err)
	}

	return regRef
}

func TestList(t *testing.T) {
	reg := registry{
		reg:     Registry(t),
		options: nil,
	}
	result, err := reg.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{
		"test-repo/test2:head",
		"test-repo/test2:latest",
		"test-repo/test:head",
		"test-repo/test:latest",
	}, result)
}
