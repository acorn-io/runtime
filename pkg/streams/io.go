package streams

import (
	"io"
	"os"
)

type Output struct {
	Out io.Writer
	Err io.Writer
}

func (o *Output) Streams() Streams {
	return Streams{
		Output: *o,
	}
}

type Streams struct {
	Output
	In io.Reader
}

func CurrentOutput() *Output {
	return &Output{
		Out: os.Stdout,
		Err: os.Stderr,
	}
}
