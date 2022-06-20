package cli

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/deployargs"
	"github.com/acorn-io/acorn/pkg/dev"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewRun() *cobra.Command {
	cmd := cli.Command(&Run{}, cobra.Command{
		Use:          "run [flags] IMAGE [deploy flags]",
		SilenceUsage: true,
		Short:        "Run an app from an app image",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Run struct {
	RunArgs
	Interactive bool `usage:"Stream logs/status in the foreground and stop on exit" short:"i"`
}

type RunArgs struct {
	Name       string   `usage:"Name of app to create" short:"n"`
	DNS        []string `usage:"Assign a friendly domain to a published container (format public:private) (ex: example.com:web)" short:"d"`
	Volume     []string `usage:"Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)" short:"v"`
	Secret     []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
	Link       []string `usage:"Link external app as a service in the current app (format app-name:service-name)" short:"l"`
	PublishAll bool     `usage:"Publish all exposed ports of application" short:"P"`
	Publish    []string `usage:"Publish exposed port of application (format [public:]private) (ex 81:80)" short:"p"`
	Profile    []string `usage:"Profile to assign default values"`
}

func (s RunArgs) ToOpts() (client.AppRunOptions, error) {
	var (
		opts client.AppRunOptions
		err  error
	)

	opts.Name = s.Name
	opts.Profiles = s.Profile

	opts.Endpoints, err = run.ParseEndpoints(s.DNS)
	if err != nil {
		return opts, err
	}

	opts.Volumes, err = run.ParseVolumes(s.Volume)
	if err != nil {
		return opts, err
	}

	opts.Secrets, err = run.ParseSecrets(s.Secret)
	if err != nil {
		return opts, err
	}

	opts.Services, err = run.ParseLinks(s.Link)
	if err != nil {
		return opts, err
	}

	opts.Ports, opts.PublishProtocols, err = run.ParsePorts(s.Publish)
	if err != nil {
		return opts, err
	}

	if s.PublishAll {
		opts.PublishProtocols = append(opts.PublishProtocols, v1.ProtocolAll)
	}

	return opts, nil
}

func (s *Run) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	opts, err := s.ToOpts()
	if err != nil {
		return err
	}

	image := args[0]
	_, flags, err := deployargs.ToFlagsFromImage(cmd.Context(), c, image)
	if err != nil {
		return err
	}

	deployParams, err := flags.Parse(args)
	if pflag.ErrHelp == err {
		return nil
	} else if err != nil {
		return err
	}

	opts.DeployArgs = deployParams

	app, err := c.AppRun(cmd.Context(), image, &opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)

	if s.Interactive {
		dev.LogLoop(cmd.Context(), c, app, nil)
		dev.AppStatusLoop(cmd.Context(), c, app)
		<-cmd.Context().Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.AppStop(ctx, app.Name)
	}

	return nil
}
