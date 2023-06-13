package streams

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type Output struct {
	Out io.Writer
	Err io.Writer
}

// MustWriteErr writes an error to o.Err and panics if it can't.
func (o *Output) MustWriteErr(err error) {
	if err == nil || o.Err == nil {
		return
	}

	if _, pErr := fmt.Fprintln(o.Err, err.Error()); pErr != nil {
		panic(pErr)
	}
}

type lockedWriter struct {
	sync.Mutex
	io.Writer
}

func (l *lockedWriter) Write(p []byte) (n int, err error) {
	if l.Writer == nil {
		return len(p), nil
	}

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
func (o *Output) Locked() *Output {
	return &Output{
		Out: &lockedWriter{Writer: o.Out},
		Err: &lockedWriter{Writer: o.Err},
	}
}

type Streams struct {
	Output
	In io.Reader
}

func Current() *Streams {
	return &Streams{
		In: os.Stdin,
		Output: Output{
			Out: os.Stdout,
			Err: os.Stderr,
		},
	}
}

func CurrentOutput() *Output {
	return &Output{
		Out: os.Stdout,
		Err: os.Stderr,
	}
}
