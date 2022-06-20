package term

import (
	"io"

	"github.com/acorn-io/acorn/pkg/streams"
	"golang.org/x/sync/errgroup"
	"k8s.io/kubectl/pkg/util/term"
)

type ExecIO struct {
	TTY      bool
	Stdin    io.WriteCloser
	Stdout   io.ReadCloser
	Stderr   io.ReadCloser
	Resize   func(Size) error
	ExitCode <-chan ExitCode
}

type Size struct {
	Height uint16
	Width  uint16
}

func IsTerminal(in io.Reader) bool {
	return term.IsTerminal(in)
}

func Pipe(execIO *ExecIO, streams *streams.Streams) (int, error) {
	if execIO.TTY {
		t := &term.TTY{
			In:  streams.In,
			Out: streams.Out,
			Raw: true,
		}
		m := t.MonitorSize(t.GetSize())
		go func() {
			for {
				size := m.Next()
				if size == nil {
					break
				}
				_ = execIO.Resize(Size{
					Height: size.Height,
					Width:  size.Width,
				})
			}
		}()
		_ = t.Safe(func() error {
			<-copyIO(execIO, streams)
			return nil
		})
	} else {
		<-copyIO(execIO, streams)
	}
	exit := <-execIO.ExitCode
	return exit.Code, exit.Err
}

func copyIO(cIO *ExecIO, streams *streams.Streams) <-chan struct{} {
	result := make(chan struct{})

	go func() {
		_, _ = io.Copy(cIO.Stdin, streams.In)
		_ = cIO.Stdin.Close()
	}()

	eg := errgroup.Group{}
	eg.Go(func() error {
		_, err := io.Copy(streams.Out, cIO.Stdout)
		return err
	})
	eg.Go(func() error {
		_, err := io.Copy(streams.Err, cIO.Stderr)
		return err
	})
	go func() {
		_ = eg.Wait()
		result <- struct{}{}
	}()

	return result
}
