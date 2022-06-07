package term

import (
	"fmt"
	"os"
	"strings"

	"github.com/acorn-io/acorn/pkg/install/progress"
	"github.com/pterm/pterm"
)

func init() {
	// Help capture text cleaner
	pterm.SetDefaultOutput(os.Stderr)
	pterm.ThemeDefault.SuccessMessageStyle = *pterm.NewStyle(pterm.FgLightGreen)
	// Customize default error.
	pterm.Success.Prefix = pterm.Prefix{
		Text:  " ✔",
		Style: pterm.NewStyle(pterm.FgLightGreen),
	}
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "    ERROR:",
		Style: pterm.NewStyle(pterm.BgLightRed, pterm.FgBlack),
	}
	pterm.Info.Prefix = pterm.Prefix{
		Text: " •",
	}
}

type Builder struct {
}

func (b *Builder) New(msg string) progress.Progress {
	return NewSpinner(msg)
}

type Spinner struct {
	spinner *pterm.SpinnerPrinter
	text    string
}

func NewSpinner(text string) *Spinner {
	spinner, err := pterm.DefaultSpinner.
		WithRemoveWhenDone(false).
		// Src: https://github.com/gernest/wow/blob/master/spin/spinners.go#L335
		WithSequence(`  ⠋ `, `  ⠙ `, `  ⠹ `, `  ⠸ `, `  ⠼ `, `  ⠴ `, `  ⠦ `, `  ⠧ `, `  ⠇ `, `  ⠏ `).
		Start(text)
	if err != nil {
		panic(err)
	}

	return &Spinner{
		spinner: spinner,
		text:    text,
	}
}

func (s *Spinner) Infof(format string, v ...interface{}) {
	s.spinner.UpdateText(fmt.Sprintf(s.text+": "+strings.TrimSpace(format), v...))
}

func (s *Spinner) Fail(err error) error {
	if err == nil {
		s.Success()
		return nil
	}
	s.spinner.Fail(err)
	return err
}

func (s *Spinner) Success() {
	s.spinner.Success(s.text)
}
