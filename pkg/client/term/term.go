package term

import (
	"io"

	"github.com/acorn-io/acorn/pkg/streams"
	"golang.org/x/sync/errgroup"
	"k8s.io/kubectl/pkg/util/term"
)

type ExecIO struct {
	Stdin    io.WriteCloser
	Stdout   io.ReadCloser
	Stderr   io.ReadCloser
	Resize   chan<- TermSize
	ExitCode <-chan ExitCode
}

type TermSize struct {
	Height uint16
	Width  uint16
}

func IsTerminal(in io.Reader) bool {
	return term.IsTerminal(in)
}

func Pipe(execIO *ExecIO, streams *streams.Streams) (int, error) {
	if term.IsTerminal(streams.In) {
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
				execIO.Resize <- TermSize{
					Height: size.Height,
					Width:  size.Width,
				}
			}
		}()
		t.Safe(func() error {
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
		io.Copy(cIO.Stdin, streams.In)
		cIO.Stdin.Close()
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
		eg.Wait()
		result <- struct{}{}
	}()

	return result
}
