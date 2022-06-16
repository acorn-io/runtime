package cli

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/dev"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/spf13/cobra"
)

func NewDev() *cobra.Command {
	cmd := cli.Command(&Dev{}, cobra.Command{
		Use:          "dev [flags] DIRECTORY",
		SilenceUsage: true,
		Short:        "Build and run an app in development mode",
		Long:         "Build and run an app in development mode",
		Args:         cobra.MaximumNArgs(1),
	})
	cmd.AddCommand(NewRender())
	return cmd
}

type Dev struct {
	File       string   `short:"f" usage:"Name of the dev file" default:"DIRECTORY/acorn.cue"`
	Name       string   `usage:"Name of app to create" short:"n"`
	DNS        []string `usage:"Assign a friendly domain to a published container (format public:private) (ex: example.com:web)" short:"d"`
	Volume     []string `usage:"Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)" short:"v"`
	Secret     []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
	Link       []string `usage:"Link external app as a service in the current app (format app-name:service-name)" short:"l"`
	PublishAll bool     `usage:"Publish all exposed ports of application" short:"P"`
	Publish    []string `usage:"Publish exposed port of application (format [public:]private) (ex 81:80)" short:"p"`
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

	volumes, err := run.ParseVolumes(s.Volume)
	if err != nil {
		return err
	}

	secrets, err := run.ParseSecrets(s.Secret)
	if err != nil {
		return err
	}

	services, err := run.ParseLinks(s.Link)
	if err != nil {
		return err
	}

	ports, publishProtocols, err := run.ParsePorts(s.Publish)
	if err != nil {
		return err
	}

	if s.PublishAll {
		publishProtocols = append(publishProtocols, v1.ProtocolAll)
	}

	return dev.Dev(cmd.Context(), s.File, &dev.Options{
		Args: args,
		Build: build.Options{
			Cwd: cwd,
		},
		Run: run.Options{
			Name:             s.Name,
			Namespace:        system.UserNamespace(),
			Client:           c,
			Endpoints:        endpoints,
			Volumes:          volumes,
			Secrets:          secrets,
			Services:         services,
			Ports:            ports,
			PublishProtocols: publishProtocols,
		},
		Log: log.Options{
			Client: c,
		},
	})
}
