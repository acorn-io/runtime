package progressbar

import (
	"errors"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/cheggaaa/pb/v3"
)

func Print(progress <-chan client.ImageProgress) error {
	var (
		err error
		bar *pb.ProgressBar
	)

	for update := range progress {
		if update.Error != "" {
			err = errors.New(update.Error)
			continue
		}

		if bar == nil {
			bar = pb.Start64(update.Total)
		}

		bar.SetCurrent(update.Complete)
	}

	bar.Finish()
	return err
}
