package cli

import (
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/dev"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/acorn-io/acorn/pkg/system"
	cli "github.com/rancher/wrangler-cli"
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
	File    string   `short:"f" usage:"Name of the dev file" default:"DIRECTORY/acorn.cue"`
	Name    string   `usage:"Name of app to create" short:"n"`
	DNS     []string `usage:"Assign a friendly domain to a published container (format public:private) (ex: example.com:web)" short:"d"`
	Volumes []string `usage:"Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)" short:"v"`
	Secrets []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
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

	endpoints, err := run.ParseEndpoints(s.DNS)
	if err != nil {
		return err
	}

	volumes, err := run.ParseVolumes(s.Volumes)
	if err != nil {
		return err
	}

	secrets, err := run.ParseSecrets(s.Secrets)
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
			Volumes:   volumes,
			Secrets:   secrets,
		},
		Log: log.Options{
			Client: c,
		},
	})
}
