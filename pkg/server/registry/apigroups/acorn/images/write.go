package images

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type ImageProgress struct {
	Total       int64  `json:"total,omitempty"`
	Complete    int64  `json:"complete,omitempty"`
	Error       string `json:"error,omitempty"`
	CurrentTask string `json:"currentTask,omitempty"`
}

type simpleUpdate struct {
	updateChan  chan ggcrv1.Update
	description string
}

func forwardUpdates(progress chan<- ImageProgress, updates chan simpleUpdate) {
	for c := range updates {
		for update := range c.updateChan {
			var errString string
			if update.Error != nil {
				errString = update.Error.Error()
			}
			progress <- ImageProgress{
				Total:       update.Total,
				Complete:    update.Complete,
				Error:       errString,
				CurrentTask: c.description,
			}
		}
	}
}

func remoteWrite(ctx context.Context, progress chan<- simpleUpdate, destRef name.Reference, source any, description string, postWriteFn func() error, opts ...remote.Option) {
	writeProgress := make(chan ggcrv1.Update)
	progress <- simpleUpdate{
		updateChan:  writeProgress,
		description: description,
	}

	var err error
	switch s := source.(type) {
	case ggcrv1.ImageIndex:
		err = remote.WriteIndex(destRef, s, append(opts, remote.WithProgress(writeProgress))...)
	case ggcrv1.Image:
		err = remote.Write(destRef, s, append(opts, remote.WithProgress(writeProgress))...)
	default:
		err = fmt.Errorf("unsupported source type: %T", source)
	}

	if err != nil {
		handleremoteWriteError(err, writeProgress)
	}
	if postWriteFn != nil {
		if err := postWriteFn(); err != nil {
			writeProgress <- ggcrv1.Update{
				Error: err,
			}
		}
	}
}

func handleremoteWriteError(err error, progress chan ggcrv1.Update) {
	if err == nil {
		return
	}
	select {
	case i, ok := <-progress:
		if ok {
			progress <- i
			progress <- ggcrv1.Update{
				Error: err,
			}
			close(progress)
		}
	default:
		progress <- ggcrv1.Update{
			Error: err,
		}
		close(progress)
	}
}
