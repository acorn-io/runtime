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
	lastMsg string
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
	msg := strings.TrimSpace(fmt.Sprintf(s.text+": "+format, v...))
	if width := pterm.GetTerminalWidth(); width > 6 && len(msg)+6 > width {
		msg = msg[:width-6]
	}
	if s.lastMsg == msg {
		return
	}
	s.lastMsg = msg
	s.spinner.UpdateText(msg)
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
