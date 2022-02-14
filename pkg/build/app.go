package build

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/streams"
)

func FromAppImage(ctx context.Context, appImage *v1.AppImage, streams streams.Output) (string, error) {
	dockerfile, err := getDockerfile()
	if err != nil {
		return "", err
	}
	defer os.Remove(dockerfile)

	buildContext, err := getContextFromAppImage(appImage)
	if err != nil {
		return "", err
	}

	io := streams.Streams()
	io.In = buildContext

	return FromBuild(ctx, "", v1.Build{
		Context:    "-",
		Dockerfile: "Dockerfile",
	}, io)
}

func getContextFromAppImage(appImage *v1.AppImage) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	tarfile := tar.NewWriter(buf)

	if err := addFile(tarfile, appdefinition.HerdCueFile, appImage.Herdfile); err != nil {
		return nil, err
	}
	if err := addFile(tarfile, appdefinition.ImageDataFile, appImage.ImageData); err != nil {
		return nil, err
	}
	if err := addFile(tarfile, "Dockerfile", []byte("FROM scratch\nCOPY . /")); err != nil {
		return nil, err
	}
	if err := addFile(tarfile, ".dockerignore", []byte("Dockerfile\n.dockerignore")); err != nil {
		return nil, err
	}
	if err := tarfile.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}

func addFile(tarfile *tar.Writer, name string, obj interface{}) error {
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

	err = tarfile.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     int64(len(data)),
		Mode:     0600,
	})
	if err != nil {
		return err
	}
	_, err = tarfile.Write(data)
	return err
}

func getDockerfile() (string, error) {
	dockerfile, err := ioutil.TempFile("", "herd-appimage-")
	if err != nil {
		return "", err
	}

	_, err = dockerfile.WriteString("FROM scratch\nCOPY . /")
	if err != nil {
		os.Remove(dockerfile.Name())
		return "", err
	}

	if err := dockerfile.Close(); err != nil {
		os.Remove(dockerfile.Name())
		return "", err
	}

	return dockerfile.Name(), nil
}
