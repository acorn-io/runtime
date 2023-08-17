package cli

import (
	"fmt"
	"net/http"
	"os"

	"github.com/acorn-io/runtime/pkg/buildserver"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"inet.af/tcpproxy"
)

func NewBuildServer(c CommandContext) *cobra.Command {
	cmd := cli.Command(&BuildServer{}, cobra.Command{
		Use:          "build-server [flags] DIRECTORY",
		Hidden:       true,
		SilenceUsage: true,
		Short:        "Run Acorn build server",
		Args:         cobra.NoArgs,
	})
	return cmd
}

type BuildServer struct {
	UUID           string `usage:"Build server BuilderUID" env:"ACORN_BUILD_SERVER_UUID"`
	Namespace      string `usage:"Build server Namespace" env:"ACORN_BUILD_SERVER_NAMESPACE"`
	PublicKey      string `usage:"Build server public key" env:"ACORN_BUILD_SERVER_PUBLIC_KEY"`
	PrivateKey     string `usage:"Build server private key" env:"ACORN_BUILD_SERVER_PRIVATE_KEY"`
	ListenPort     int    `usage:"HTTP listen port" env:"ACORN_BUILD_SERVER_PORT" default:"8080"`
	ForwardPort    int    `usage:"Forward TCP Listen Port" default:"5000"`
	ForwardService string `usage:"Forwarding Address" env:"ACORN_BUILD_SERVER_FORWARD_SERVICE"`
}

func (s *BuildServer) Run(cmd *cobra.Command, _ []string) error {
	c, err := k8sclient.Default()
	if err != nil {
		return err
	}

	pubKey, privKey, err := buildserver.ToKeys(s.PublicKey, s.PrivateKey)
	if err != nil {
		return err
	}

	server := buildserver.NewServer(s.UUID, s.Namespace, pubKey, privKey, c)
	address := fmt.Sprintf("0.0.0.0:%d", s.ListenPort)

	var p *tcpproxy.Proxy
	if s.ForwardService != "" {
		p = new(tcpproxy.Proxy)
		p.AddRoute(fmt.Sprintf(":%d", s.ForwardPort), tcpproxy.To(s.ForwardService))
		go func() {
			logrus.Infof("Forwarding :%d to %s", s.ForwardPort, s.ForwardService)
			logrus.Fatal(p.Run())
		}()
	}

	go func() {
		// Nothing here is using the context. Therefore, when the context is canceled, we should just exit.
		<-cmd.Context().Done()
		if p != nil {
			if err = p.Close(); err != nil {
				logrus.Errorf("Error closing proxy: %v", err)
			}
		}
		os.Exit(0)
	}()

	logrus.Infof("Listening on %s", address)
	return http.ListenAndServe(address, server)
}
