package helper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/build"
)

func HerdImages(ctx context.Context) (*v1.ImagesData, error) {
	herdCue, err := findHerdCue()
	if err != nil {
		return nil, err
	}
	image, err := build.Build(ctx, herdCue, &build.Options{
		Cwd: filepath.Dir(herdCue),
	})
	if err != nil {
		return nil, err
	}
	return &image.ImageData, nil
}

func findHerdCue() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return traverse(dir)
}

func traverse(dir string) (string, error) {
	herdCue := filepath.Join(dir, "herd.cue")
	_, err := os.Stat(herdCue)
	if os.IsNotExist(err) {
		pwd := filepath.Dir(dir)
		if dir == pwd {
			return "", fmt.Errorf("failed to find herd.cue")
		}
		return traverse(pwd)
	} else if err != nil {
		return "", err
	}
	return herdCue, nil
}
