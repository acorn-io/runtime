package progressbar

import (
	"errors"
	"fmt"
	"time"

	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/pterm/pterm"
)

func Print(progress <-chan client.ImageProgress) error {
	var (
		err error
		bar *pterm.ProgressbarPrinter
	)

	if pterm.RawOutput {
		var last client.ImageProgress
		for update := range typed.Every(time.Second, progress) {
			if update.Error != "" {
				err = errors.New(update.Error)
				continue
			}
			if update.Total == 0 {
				continue
			}
			if update == last {
				continue
			}
			fmt.Printf("[%d/%d]\n", update.Complete, update.Total)
			last = update
		}
		if last.Total != 0 && last.Total != last.Complete {
			fmt.Printf("[%d/%d]\n", last.Total, last.Total)
		}
	} else {
		var currentTask string
		for update := range progress {
			if update.Error != "" {
				err = errors.New(update.Error)
				continue
			}

			if update.Total == 0 {
				// we need total to properly print status
				continue
			}

			if update.CurrentTask != "" && update.CurrentTask != currentTask {
				if bar != nil {
					bar.Add(bar.Total - bar.Current)
					_, _ = bar.Stop()
					bar = nil
				}
				currentTask = update.CurrentTask
			}

			if bar == nil {
				bar = pterm.DefaultProgressbar.
					WithTotal(int(update.Total)).
					WithCurrent(int(update.Complete))
				if currentTask != "" {
					bar = bar.WithTitle(currentTask)
				}
				bar, _ = bar.Start()
			}

			if int(update.Complete) > bar.Current {
				bar.Add(int(update.Complete) - bar.Current)
			}
		}

		if bar != nil {
			if err == nil && bar.Current != bar.Total {
				bar.Add(bar.Total - bar.Current)
			}
			_, _ = bar.Stop()
		}
	}

	return err
}
