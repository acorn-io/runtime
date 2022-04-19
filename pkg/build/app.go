package build

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/containerd/containerd/platforms"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/streams"
)

func FromAppImage(ctx context.Context, namespace string, appImage *v1.AppImage, streams streams.Output) (string, error) {
	tempContext, err := getContextFromAppImage(appImage)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempContext)

	io := streams.Streams()
	tag, err := buildImageNoManifest(ctx, tempContext, namespace, v1.Platform(platforms.DefaultSpec()), v1.Build{
		Context:    ".",
		Dockerfile: "Dockerfile",
	}, io)
	if err != nil {
		return "", err
	}

	return createAppManifest(ctx, tag, appImage.ImageData)
}

func getContextFromAppImage(appImage *v1.AppImage) (_ string, err error) {
	tempDir, err := ioutil.TempDir("", "herd-app-image-context")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tempDir)
		}
	}()

	imageData, err := digestOnly(appImage.ImageData)
	if err != nil {
		return "", err
	}

	if err := addFile(tempDir, appdefinition.HerdCueFile, appImage.Herdfile); err != nil {
		return "", err
	}
	if err := addFile(tempDir, appdefinition.ImageDataFile, imageData); err != nil {
		return "", err
	}
	if err := addFile(tempDir, "Dockerfile", "FROM scratch\nCOPY . /"); err != nil {
		return "", err
	}
	if err := addFile(tempDir, ".dockerignore", "Dockerfile\n.dockerignore"); err != nil {
		return "", err
	}
	if len(appImage.BuildParams) > 0 {
		if err := addFile(tempDir, "build.json", appImage.BuildParams); err != nil {
			return "", err
		}
	}
	return tempDir, nil
}

func addFile(tempDir, name string, obj interface{}) error {
	var (
		data []byte
		err  error
	)
	if d, ok := obj.([]byte); ok {
		data = d
	} else if s, ok := obj.(string); ok {
		data = []byte(s)
	} else {
		data, err = json.Marshal(obj)
		if err != nil {
			return err
		}
	}

	target := filepath.Join(tempDir, name)
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}

	err = ioutil.WriteFile(target, data, 0600)
	if err != nil {
		return err
	}

	return os.Chtimes(target, time.Unix(0, 0), time.Unix(0, 0))
}
