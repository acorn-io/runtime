package build

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/streams"
)

func FromAppImage(ctx context.Context, appImage *v1.AppImage, streams streams.Output) (string, error) {
	tempContext, err := getContextFromAppImage(appImage)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempContext)

	io := streams.Streams()
	return FromBuild(ctx, tempContext, v1.Build{}, io)
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

	if err := addFile(tempDir, appdefinition.HerdCueFile, []byte(appImage.Herdfile)); err != nil {
		return "", err
	}
	if err := addFile(tempDir, appdefinition.ImageDataFile, appImage.ImageData); err != nil {
		return "", err
	}
	if err := addFile(tempDir, "Dockerfile", []byte("FROM scratch\nCOPY . /")); err != nil {
		return "", err
	}
	if err := addFile(tempDir, ".dockerignore", []byte("Dockerfile\n.dockerignore")); err != nil {
		return "", err
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

	return ioutil.WriteFile(target, data, 0600)
}
