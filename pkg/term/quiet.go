package term

import (
	"github.com/acorn-io/runtime/pkg/install/progress"
	"github.com/sirupsen/logrus"
)

type QuietBuilder struct {
}

func (b *QuietBuilder) New(msg string) progress.Progress {
	return newQuietSpinner(msg)
}

type quietSpinner struct {
	text string
}

func newQuietSpinner(text string) *quietSpinner {
	return &quietSpinner{
		text: text,
	}
}

func (s *quietSpinner) Fail(err error) error {
	if err != nil {
		logrus.Errorf("Error encountered during '%v': %v", s.text, err)
	}
	return err
}

func (*quietSpinner) Infof(string, ...any) {
	// No-op, we're being quiet
}

func (*quietSpinner) SuccessWithWarning(string, ...any) {
	// No-op, we're being quiet
}

func (*quietSpinner) Success() {
	// No-op, we're being quiet
}
