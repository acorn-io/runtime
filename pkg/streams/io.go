package streams

import (
	"io"
	"os"
	"sync"
)

type Output struct {
	Out io.Writer
	Err io.Writer
}

type lockedWriter struct {
	sync.Mutex
	io.Writer
}

func (l *lockedWriter) Write(p []byte) (n int, err error) {
	l.Lock()
	defer l.Unlock()
	return l.Writer.Write(p)
}

func (o *Output) Streams() Streams {
	return Streams{
		Output: *o,
	}
}

// Locked with wrap both Out and Err with a Mutex to make it safe for concurrent access
func (o *Output) Locked() Output {
	return Output{
		Out: &lockedWriter{Writer: o.Out},
		Err: &lockedWriter{Writer: o.Err},
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
