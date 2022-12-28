package build

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/buildclient"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type AppImageOptions struct {
	FullTag       bool
	RemoteOptions []remote.Option
	Keychain      authn.Keychain
}

func (a *AppImageOptions) GetFullTag() bool {
	if a == nil {
		return false
	}
	return a.FullTag
}

func (a *AppImageOptions) GetKeychain() authn.Keychain {
	if a == nil {
		return nil
	}
	return a.Keychain
}

func (a *AppImageOptions) GetRemoteOptions() []remote.Option {
	if a == nil {
		return nil
	}
	return a.RemoteOptions
}

func FromAppImage(ctx context.Context, pushRepo string, appImage *v1.AppImage, messages buildclient.Messages, opts *AppImageOptions) (string, error) {
	tempContext, err := getContextFromAppImage(appImage)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempContext)

	tag, err := buildImageNoManifest(ctx, pushRepo, tempContext, v1.Build{
		Context:    ".",
		Dockerfile: "Dockerfile",
	}, messages, opts.GetKeychain())
	if err != nil {
		return "", err
	}

	return createAppManifest(ctx, tag, appImage.ImageData, opts.GetFullTag(), opts.GetRemoteOptions())
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
