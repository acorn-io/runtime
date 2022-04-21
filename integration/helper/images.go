package helper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
)

func AcornImages(ctx context.Context) (*v1.ImagesData, error) {
	acornCue, err := findAcornCue()
	if err != nil {
		return nil, err
	}
	image, err := build.Build(ctx, acornCue, &build.Options{
		Cwd: filepath.Dir(acornCue),
	})
	if err != nil {
		return nil, err
	}
	return &image.ImageData, nil
}

func findAcornCue() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return traverse(dir)
}

func traverse(dir string) (string, error) {
	acornCue := filepath.Join(dir, "acorn.cue")
	_, err := os.Stat(acornCue)
	if os.IsNotExist(err) {
		pwd := filepath.Dir(dir)
		if dir == pwd {
			return "", fmt.Errorf("failed to find acorn.cue")
		}
		return traverse(pwd)
	} else if err != nil {
		return "", err
	}
	return acornCue, nil
}
