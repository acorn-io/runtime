package cli

import (
	"fmt"
	"time"

	"github.com/goombaio/namegenerator"
	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/ibuildthecloud/herd/pkg/run"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

var (
	nameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

func NewRun() *cobra.Command {
	return cli.Command(&Run{}, cobra.Command{
		Use:          "run [flags] IMAGE",
		SilenceUsage: true,
		Short:        "Run an app from an app image",
		Args:         cobra.RangeArgs(1, 1),
	})
}

type Run struct {
	Name        string   `usage:"Name of app to create" short:"n"`
	Endpoint    []string `usage:"Bind a published host to a friendly domain (format public:private) (ex: example.com:web)" short:"b"`
	PullSecrets []string `usage:"Secret names to authenticate pull images in cluster" short:"l"`
}

func (s *Run) getName() (string, bool) {
	if s.Name != "" {
		return "", false
	}
	return nameGenerator.Generate(), true

}

func (s *Run) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	opts := client.AppRunOptions{
		Name:             s.Name,
		ImagePullSecrets: s.PullSecrets,
	}

	opts.Endpoints, err = run.ParseEndpoints(s.Endpoint)
	if err != nil {
		return err
	}

	app, err := c.AppRun(cmd.Context(), args[0], &opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)
	return nil
}
