package progress

import (
	"encoding/json"
	"fmt"
	"io"
)

func NewStream(out io.Writer) Builder {
	return &stream{out: out}
}

type stream struct {
	out io.Writer
}

func (s *stream) New(component string) Progress {
	return &streamProgress{
		out:       s.out,
		component: component,
	}
}

type message struct {
	Error     bool   `json:"error,omitempty"`
	Done      bool   `json:"done,omitempty"`
	Component string `json:"component,omitempty"`
	Message   string `json:"message,omitempty"`
}

type streamProgress struct {
	out       io.Writer
	component string
}

func (s *streamProgress) write(isError, isDone bool, msg string) {
	_ = json.NewEncoder(s.out).Encode(&message{
		Error:     isError,
		Done:      isDone,
		Component: s.component,
		Message:   msg,
	})
}

func (s *streamProgress) Infof(fmtStr string, args ...any) {
	s.write(false, false, fmt.Sprintf(fmtStr, args...))
}

func (s *streamProgress) Fail(err error) error {
	if err == nil {
		s.Success()
		return nil
	}
	s.write(true, true, err.Error())
	return err
}

func (s *streamProgress) SuccessWithWarning(fmtStr string, args ...any) {
	s.write(false, true, fmt.Sprintf(fmtStr, args...))
}

func (s *streamProgress) Success() {
	s.write(false, true, "")
}
