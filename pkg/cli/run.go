package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/deployargs"
	"github.com/acorn-io/acorn/pkg/dev"
	"github.com/acorn-io/acorn/pkg/rulerequest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

func NewRun(out io.Writer) *cobra.Command {
	cmd := cli.Command(&Run{out: out}, cobra.Command{
		Use:          "run [flags] IMAGE|DIRECTORY [acorn args]",
		SilenceUsage: true,
		Short:        "Run an app from an image or Acornfile",
	})
	cmd.PersistentFlags().Lookup("dangerous").Hidden = true
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Run struct {
	RunArgs
	Interactive       bool `usage:"Enable interactive dev mode: build image, stream logs/status in the foreground and stop on exit" short:"i" name:"dev"`
	BidirectionalSync bool `usage:"In interactive mode download changes in addition to uploading" short:"b"`

	out io.Writer
}

type RunArgs struct {
	Name        string   `usage:"Name of app to create" short:"n"`
	File        string   `short:"f" usage:"Name of the build file" default:"DIRECTORY/Acornfile"`
	Volume      []string `usage:"Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)" short:"v" split:"false"`
	Secret      []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
	Link        []string `usage:"Link external app as a service in the current app (format app-name:service-name)" short:"l"`
	PublishAll  *bool    `usage:"Publish all (true) or none (false) of the defined ports of application" short:"P"`
	Publish     []string `usage:"Publish port of application (format [public:]private) (ex 81:80)" short:"p"`
	Expose      []string `usage:"In cluster expose ports of an application (format [public:]private) (ex 81:80)"`
	Profile     []string `usage:"Profile to assign default values"`
	Env         []string `usage:"Environment variables to set on running containers" short:"e"`
	Labels      []string `usage:"Add labels" short:"L"`
	Annotations []string `usage:"Add annotations" short:"a"`
	Dangerous   bool     `usage:"Automatically approve all privileges requested by the application"`
	Output      string   `usage:"Output API request without creating app (json, yaml)" short:"o"`
}

func (s RunArgs) ToOpts() (client.AppRunOptions, error) {
	var (
		opts client.AppRunOptions
		err  error
	)

	opts.Name = s.Name
	opts.Profiles = s.Profile

	opts.Volumes, err = v1.ParseVolumes(s.Volume, true)
	if err != nil {
		return opts, err
	}

	opts.Secrets, err = v1.ParseSecrets(s.Secret)
	if err != nil {
		return opts, err
	}

	opts.Links, err = v1.ParseLinks(s.Link)
	if err != nil {
		return opts, err
	}

	opts.Env = v1.ParseNameValues(true, s.Env...)

	opts.Labels = v1.ParseNameValuesToMap(s.Labels...)
	opts.Annotations = v1.ParseNameValuesToMap(s.Annotations...)

	opts.Ports, err = v1.ParsePortBindings(true, s.Publish)
	if err != nil {
		return opts, err
	}

	expose, err := v1.ParsePortBindings(false, s.Expose)
	if err != nil {
		return opts, err
	}
	opts.Ports = append(opts.Ports, expose...)

	if s.PublishAll != nil && *s.PublishAll {
		opts.PublishMode = v1.PublishModeAll
	} else if s.PublishAll != nil && !*s.PublishAll {
		opts.PublishMode = v1.PublishModeNone
	}

	return opts, nil
}

func isDirectory(cwd string) (bool, error) {
	if s, err := os.Stat(cwd); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else if !s.IsDir() {
		return false, fmt.Errorf("%s is not a directory", cwd)
	}
	return true, nil
}

func buildImage(ctx context.Context, file, cwd string, args, profiles []string) (string, error) {
	params, err := build.ParseParams(file, cwd, args)
	if err != nil {
		return "", err
	}

	image, err := build.Build(ctx, file, &build.Options{
		Cwd:      cwd,
		Args:     params,
		Profiles: profiles,
	})
	if err != nil {
		return "", err
	}

	return image.ID, nil
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

	cwd := "."
	if len(args) > 0 {
		cwd = args[0]
	}

	isDir, err := isDirectory(cwd)
	if err != nil {
		return err
	}

	if s.Interactive && isDir {
		// Force install prompt if needed
		_, _ = c.Info(cmd.Context())

		return dev.Dev(cmd.Context(), s.File, &dev.Options{
			Args:   args,
			Client: c,
			Build: build.Options{
				Cwd:      cwd,
				Profiles: opts.Profiles,
			},
			Run:               opts,
			Dangerous:         s.Dangerous,
			BidirectionalSync: s.BidirectionalSync,
		})
	}

	if s.Interactive {
		s.Profile = append([]string{"dev?"}, s.Profile...)
	}

	image := cwd
	if isDir {
		image, err = buildImage(cmd.Context(), s.File, cwd, args, s.Profile)
		if err == pflag.ErrHelp {
			return nil
		} else if err != nil {
			return err
		}
	}

	if len(args) > 1 {
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
	}

	if s.Output != "" {
		app := client.ToApp(c.GetNamespace(), image, &opts)
		return outputApp(s.out, s.Output, app)
	}

	app, err := rulerequest.PromptRun(cmd.Context(), c, s.Dangerous, image, opts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)

	if s.Interactive {
		go func() { _ = dev.LogLoop(cmd.Context(), c, app, nil) }()
		go func() { _ = dev.AppStatusLoop(cmd.Context(), c, app) }()
		<-cmd.Context().Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.AppStop(ctx, app.Name)
	}

	return nil
}

func outputApp(out io.Writer, format string, app *apiv1.App) error {
	data, err := json.Marshal(app)
	if err != nil {
		return err
	}

	mapData := map[string]interface{}{}
	if err := json.Unmarshal(data, &mapData); err != nil {
		return err
	}

	delete(mapData, "status")
	delete(mapData["metadata"].(map[string]interface{}), "uid")
	delete(mapData["metadata"].(map[string]interface{}), "resourceVersion")
	delete(mapData["metadata"].(map[string]interface{}), "managedFields")
	delete(mapData["metadata"].(map[string]interface{}), "creationTimestamp")

	if format == "json" {
		data, err = json.MarshalIndent(mapData, "", "  ")
	} else {
		data, err = yaml.Marshal(mapData)
	}
	if err != nil {
		return err
	}

	if out == nil {
		_, err = os.Stdout.Write(data)
	} else {
		_, err = out.Write(data)
	}
	return err
}
