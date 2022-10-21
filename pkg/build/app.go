package build

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/streams"
)

type AppImageOptions struct {
	FullTag bool
}

func FromAppImage(ctx context.Context, c client.Client, appImage *v1.AppImage, streams streams.Output, opts *AppImageOptions) (string, error) {
	tempContext, err := getContextFromAppImage(appImage)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempContext)

	io := streams.Streams()
	tag, err := buildImageNoManifest(ctx, c, tempContext, v1.Build{
		Context:    ".",
		Dockerfile: "Dockerfile",
	}, io)
	if err != nil {
		return "", err
	}

	var fullTag bool
	if opts != nil {
		fullTag = opts.FullTag
	}

	return createAppManifest(ctx, c, tag, appImage.ImageData, fullTag)
}

func getContextFromAppImage(appImage *v1.AppImage) (_ string, err error) {
	tempDir, err := os.MkdirTemp("", "acorn-app-image-context")
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

	if err := addFile(tempDir, appdefinition.AcornCueFile, appImage.Acornfile); err != nil {
		return "", err
	}
	if err := addFile(tempDir, appdefinition.ImageDataFile, imageData); err != nil {
		return "", err
	}
	if err := addFile(tempDir, appdefinition.VCSDataFile, appImage.VCS); err != nil {
		return "", err
	}
	if err := addFile(tempDir, "Dockerfile", "FROM scratch\nCOPY . /"); err != nil {
		return "", err
	}
	if err := addFile(tempDir, ".dockerignore", "Dockerfile\n.dockerignore"); err != nil {
		return "", err
	}
	if len(appImage.BuildArgs) > 0 {
		if err := addFile(tempDir, appdefinition.BuildDataFile, appImage.BuildArgs); err != nil {
			return "", err
		}
	}
	return tempDir, nil
}

func addFile(tempDir, name string, obj any) error {
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

	return os.WriteFile(target, data, 0600)
}
