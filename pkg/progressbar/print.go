package progressbar

import (
	"errors"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/pterm/pterm"
)

func Print(progress <-chan client.ImageProgress) error {
	var (
		err error
		bar *pterm.ProgressbarPrinter
	)

	for update := range progress {
		if update.Error != "" {
			err = errors.New(update.Error)
			continue
		}

		if bar == nil {
			bar, _ = pterm.DefaultProgressbar.
				WithTotal(int(update.Total)).
				WithCurrent(int(update.Complete)).Start()
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

	return err
}
