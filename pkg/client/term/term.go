package term

import (
	"io"
	"time"

	"github.com/acorn-io/runtime/pkg/streams"
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
		c, err := io.Copy(cIO.Stdin, streams.In)
		if c == 0 && err == nil {
			// Very good chance the stdin was closed at start, so just wait
			// until stdout/stderr are done
			<-result
		} else {
			// This is an unfortunate hack. It does not seem possible to close the
			// stdin side of an exec session to kubernetes over a WebSocket. This
			// means that for a command like "echo hi | acorn exec container cat" we
			// can not reliably run it. For a command like that you have to finish
			// reading stdin, close it, and then fully read the response.  If you
			// don't close stdin, the "cat" command will not exit.  If you do
			// don't fully read the response you lose the output. So here we just
			// sleep one second hoping that is enough time to read the output
			time.Sleep(1 * time.Second)
		}
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
		close(result)
	}()

	return result
}
