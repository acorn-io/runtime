package build

import (
	"encoding/json"
	"os"
	"path/filepath"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/appdefinition"
	"github.com/acorn-io/runtime/pkg/version"
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

func fromAppImage(ctx *buildContext, dataFiles appdefinition.DataFiles, appImage *v1.AppImage) (string, error) {
	tempContext, err := getContextFromAppImage(dataFiles, appImage)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempContext)

	tag, err := buildImageNoManifest(ctx, tempContext, v1.Build{
		Context:    ".",
		Dockerfile: "Dockerfile",
	})
	if err != nil {
		return "", err
	}

	return createAppManifest(tag, appImage.ImageData, ctx.remoteOpts)
}

func getContextFromAppImage(dataFiles appdefinition.DataFiles, appImage *v1.AppImage) (_ string, err error) {
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

	if len(dataFiles.Icon) > 0 {
		if err := addFile(tempDir, appdefinition.IconFile+dataFiles.IconSuffix, dataFiles.Icon); err != nil {
			return "", err
		}
	}

	if len(dataFiles.Readme) > 0 {
		if err := addFile(tempDir, appdefinition.ReadmeFile, dataFiles.Readme); err != nil {
			return "", err
		}
	}

	if err := addFile(tempDir, appdefinition.Acornfile, appImage.Acornfile); err != nil {
		return "", err
	}
	if err := addFile(tempDir, appdefinition.ImageDataFile, imageData); err != nil {
		return "", err
	}
	if err := addFile(tempDir, appdefinition.VCSDataFile, appImage.VCS); err != nil {
		return "", err
	}
	if err := addFile(tempDir, appdefinition.VersionFile, v1.AppImageVersion{
		RuntimeVersion:  version.Get().String(),
		AcornfileSchema: appdefinition.AcornfileSchemaVersion,
	}); err != nil {
		return "", err
	}
	if err := addFile(tempDir, "Dockerfile", "FROM scratch\nCOPY . /"); err != nil {
		return "", err
	}
	if err := addFile(tempDir, ".dockerignore", "Dockerfile\n.dockerignore"); err != nil {
		return "", err
	}
	if len(appImage.BuildArgs.GetData()) > 0 {
		if err := addFile(tempDir, appdefinition.BuildDataFile, appImage.BuildArgs); err != nil {
			return "", err
		}
	}
	if err := addFile(tempDir, appdefinition.BuildContextFile, appImage.BuildContext); err != nil {
		return "", err
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
