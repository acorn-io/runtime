package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/flagparams"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/rancher/wrangler-cli"
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
	Image   string   `json:"image,omitempty"`
	DNS     []string `usage:"Assign a friendly domain to a published container (format public:private) (ex: example.com:web)" short:"d"`
	Volumes []string `usage:"Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)" short:"v"`
	Secrets []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
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

	app, err := c.AppUpdate(cmd.Context(), name, &opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)
	return nil
}
