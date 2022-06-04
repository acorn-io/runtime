package cli

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/flagparams"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/acorn-io/baaah/pkg/typed"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
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
	Name   string   `usage:"Name of app to create" short:"n"`
	DNS    []string `usage:"Assign a friendly domain to a published container (format public:private) (ex: example.com:web)" short:"d"`
	Volume []string `usage:"Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)" short:"v"`
	Secret []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
	Link   []string `usage:"Link external app as a service in the current app (format app-name:service-name)" short:"l"`
}

func usage(app *v1.AppSpec) func() {
	return func() {
		fmt.Println()
		if len(app.Volumes) == 0 {
			fmt.Println("Volumes:   <none>")
		} else {
			fmt.Print("Volumes:   ")
			fmt.Println(strings.Join(typed.Keys(app.Volumes), ", "))
		}

		if len(app.Secrets) == 0 {
			fmt.Println("Secrets:   <none>")
		} else {
			fmt.Print("Secrets:   ")
			fmt.Println(strings.Join(typed.Keys(app.Secrets), ", "))
		}

		if len(app.Secrets) == 0 {
			fmt.Println("Container: <none>")
		} else {
			fmt.Print("Container: ")
			fmt.Println(strings.Join(typed.Keys(app.Containers), ", "))
		}

		var ports []string
		for containerName, container := range app.Containers {
			for _, port := range container.Ports {
				if port.Publish {
					ports = append(ports, fmt.Sprintf("%s:%d/%s", containerName, port.Port, port.Protocol))
				}
			}
			for _, sidecar := range container.Sidecars {
				for _, port := range sidecar.Ports {
					if port.Publish {
						ports = append(ports, fmt.Sprintf("%s:%d/%s", containerName, port.Port, port.Protocol))
					}
				}
			}
		}
		sort.Strings(ports)

		if len(ports) == 0 {
			fmt.Println("Ports:     <none>")
		} else {
			fmt.Print("Ports:     ")
			fmt.Println(strings.Join(ports, ", "))
		}

		fmt.Println()
	}
}

func (s *Run) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	image := args[0]

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

	opts := client.AppRunOptions{
		Name:         s.Name,
		DeployParams: deployParams,
	}

	opts.Endpoints, err = run.ParseEndpoints(s.DNS)
	if err != nil {
		return err
	}

	opts.Volumes, err = run.ParseVolumes(s.Volume)
	if err != nil {
		return err
	}

	opts.Secrets, err = run.ParseSecrets(s.Secret)
	if err != nil {
		return err
	}

	opts.Services, err = run.ParseLinks(s.Link)
	if err != nil {
		return err
	}

	app, err := c.AppRun(cmd.Context(), image, &opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)
	return nil
}
