package cli

import (
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/flagparams"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewUpdate() *cobra.Command {
	cmd := cli.Command(&Update{}, cobra.Command{
		Use:          "update [flags] APP_NAME [deploy flags]",
		SilenceUsage: true,
		Short:        "Update a deployed app",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Update struct {
	Image      string   `json:"image,omitempty"`
	DNS        []string `usage:"Assign a friendly domain to a published container (format public:private) (ex: example.com:web)" short:"d"`
	Volumes    []string `usage:"Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)" short:"v"`
	Secrets    []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
	Link       []string `usage:"Link external app as a service in the current app (format app-name:service-name)" short:"l"`
	PublishAll *bool    `usage:"Publish all exposed ports of application" short:"P"`
	Publish    []string `usage:"Publish exposed port of application (format [public:]private) (ex 81:80)" short:"p"`
}

func (s *Update) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	name := args[0]
	image := s.Image

	if image == "" {
		app, err := c.AppGet(cmd.Context(), name)
		if err != nil {
			return err
		}
		image = app.Spec.Image
	}

	imageDetails, err := c.ImageDetails(cmd.Context(), image, nil)
	if err != nil {
		return err
	}

	appDef, err := appdefinition.FromAppImage(&imageDetails.AppImage)
	if err != nil {
		return err
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		return err
	}

	params, err := appDef.DeployParams()
	if err != nil {
		return err
	}

	flags := flagparams.New(image, params)
	flags.Usage = usage(appSpec)

	deployParams, err := flags.Parse(args)
	if pflag.ErrHelp == err {
		return nil
	} else if err != nil {
		return err
	}

	opts := client.AppUpdateOptions{
		Image:        image,
		DeployParams: deployParams,
	}

	opts.Endpoints, err = run.ParseEndpoints(s.DNS)
	if err != nil {
		return err
	}

	opts.Volumes, err = run.ParseVolumes(s.Volumes)
	if err != nil {
		return err
	}

	opts.Secrets, err = run.ParseSecrets(s.Secrets)
	if err != nil {
		return err
	}

	opts.Services, err = run.ParseLinks(s.Link)
	if err != nil {
		return err
	}

	opts.Ports, opts.PublishProtocols, err = run.ParsePorts(s.Publish)
	if err != nil {
		return err
	}

	if s.PublishAll != nil {
		if *s.PublishAll {
			opts.PublishProtocols = append(opts.PublishProtocols, v1.ProtocolAll)
		} else {
			opts.PublishProtocols = append(opts.PublishProtocols, v1.ProtocolNone)
		}
	}

	app, err := c.AppUpdate(cmd.Context(), name, &opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)
	return nil
}
