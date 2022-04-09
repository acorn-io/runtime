package cli

import (
	"fmt"
	"time"

	"github.com/goombaio/namegenerator"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/run"
	"github.com/ibuildthecloud/herd/pkg/system"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

var (
	nameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

func NewRun() *cobra.Command {
	return cli.Command(&Run{}, cobra.Command{
		Use:          "run [flags] IMAGE",
		SilenceUsage: true,
		Short:        "Run an app from an app image",
		Long:         "Run all dependent container and app images from your herd.cue file",
		Args:         cobra.RangeArgs(1, 1),
	})
}

type Run struct {
	Name     string   `usage:"Name of app to create" short:"n"`
	Endpoint []string `usage:"Bind a published host to a friendly domain (format public:private) (ex: example.com:web)" short:"b"`
}

func (s *Run) getName() (string, bool) {
	if s.Name != "" {
		return "", false
	}
	return nameGenerator.Generate(), true

}

func (s *Run) Run(cmd *cobra.Command, args []string) error {
	var (
		lastErr error
		app     *v1.AppInstance
		image   = args[0]
		opts    = &run.Options{
			Name:      s.Name,
			Namespace: system.UserNamespace(),
		}
	)

	opts.Endpoints, lastErr = run.ParseEndpoints(s.Endpoint)
	if lastErr != nil {
		return lastErr
	}

	for i := 0; i < 3; i++ {
		app, lastErr = run.Run(cmd.Context(), image, opts)
		if lastErr == nil {
			fmt.Println(app.Name)
			return nil
		}
		if apierror.IsAlreadyExists(lastErr) && opts.Name == "" {
			continue
		} else {
			return lastErr
		}
	}

	return fmt.Errorf("after three tried failed to create app: %w", lastErr)
}
