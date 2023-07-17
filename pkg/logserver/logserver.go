// Adapted from https://github.com/rancher/log/blob/52031d45f5fdb71cecb9a314865624b42514dbed/server/server.go
// The only real differences here are that we are using logrus directly instead of a wrapper around it,
// and that we are using an abstract namespace (non-file) socket.
// This is only compatible with our fork of the client: https://github.com/acorn-io/loglevel

package logserver

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	DefaultSocketLocation = "\x00logserver"
)

// Server structure is used to the store backend information
type Server struct {
	SocketLocation string
	Debug          bool
}

func init() {
	go StartServerWithDefaults()
}

// StartServerWithDefaults starts the server with default values
func StartServerWithDefaults() {
	logrus.SetLevel(logrus.InfoLevel)
	s := Server{
		SocketLocation: DefaultSocketLocation,
	}
	s.Start()
}

// Start the server
func (s *Server) Start() {
	_ = os.Remove(s.SocketLocation)
	go func() {
		_ = s.ListenAndServe()
	}()
}

// ListenAndServe is used to setup handlers and
// start listening on the specified location
func (s *Server) ListenAndServe() error {
	logrus.Infof("Listening on %s", s.SocketLocation)
	server := http.Server{}
	http.HandleFunc("/v1/loglevel", s.loglevel)
	socketListener, err := net.Listen("unix", s.SocketLocation)
	if err != nil {
		return err
	}
	return server.Serve(socketListener)
}

func (s *Server) loglevel(rw http.ResponseWriter, req *http.Request) {
	// curl -X POST -d "level=debug" localhost:12345/v1/loglevel
	logrus.Debugf("Received loglevel request")
	if req.Method == http.MethodGet {
		level := logrus.GetLevel().String()
		_, _ = rw.Write([]byte(fmt.Sprintf("%s\n", level)))
	}

	if req.Method == http.MethodPost {
		if err := req.ParseForm(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Failed to parse form: %v\n", err)))
		}
		level, err := logrus.ParseLevel(req.Form.Get("level"))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Failed to parse loglevel: %v\n", err)))
		} else {
			logrus.SetLevel(level)
			_, _ = rw.Write([]byte("OK\n"))
		}
	}
}
