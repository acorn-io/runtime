package cli

import (
	"github.com/ibuildthecloud/herd/pkg/build"
	"github.com/ibuildthecloud/herd/pkg/dev"
	hclient "github.com/ibuildthecloud/herd/pkg/k8sclient"
	"github.com/ibuildthecloud/herd/pkg/log"
	"github.com/ibuildthecloud/herd/pkg/run"
	"github.com/ibuildthecloud/herd/pkg/system"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewDev() *cobra.Command {
	return cli.Command(&Dev{}, cobra.Command{
		Use:          "dev [flags] DIRECTORY",
		SilenceUsage: true,
		Short:        "Build and run an app in development mode",
		Long:         "Build and run an app in development mode",
		Args:         cobra.MaximumNArgs(1),
	})
}

type Dev struct {
	File     string   `short:"f" desc:"Name of the dev file" default:"DIRECTORY/herd.cue"`
	Name     string   `usage:"Name of app to create" short:"n"`
	Endpoint []string `usage:"Bind a published host to a friendly domain (format public:private) (ex: example.com:web)" short:"b"`
}

func (s *Dev) Run(cmd *cobra.Command, args []string) error {
	cwd := "."
	if len(args) > 0 {
		cwd = args[0]
	}

	c, err := hclient.Default()
	if err != nil {
		return err
	}

	endpoints, err := run.ParseEndpoints(s.Endpoint)
	if err != nil {
		return err
	}

	return dev.Dev(cmd.Context(), s.File, &dev.Options{
		Build: build.Options{
			Cwd: cwd,
		},
		Run: run.Options{
			Name:      s.Name,
			Namespace: system.UserNamespace(),
			Client:    c,
			Endpoints: endpoints,
		},
		Log: log.Options{
			Client: c,
		},
	})
}
