package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/build"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/deployargs"
	"github.com/acorn-io/acorn/pkg/rulerequest"
	"github.com/acorn-io/acorn/pkg/wait"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

func NewRun(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Run{out: c.StdOut, client: c.ClientFactory}, cobra.Command{
		Use:               "run [flags] IMAGE|DIRECTORY [acorn args]",
		SilenceUsage:      true,
		Short:             "Run an app from an image or Acornfile",
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).withSuccessDirective(cobra.ShellCompDirectiveDefault).withShouldCompleteOptions(onlyNumArgs(1)).complete,
		Example: `# Publish and Expose Port Syntax
  # Publish port 80 for any containers that define it as a port
  acorn run -p 80 .

  # Publish container "myapp" using the hostname app.example.com
  acorn run --publish app.example.com:myapp .

  # Expose port 80 to the rest of the cluster as port 8080
  acorn run --expose 8080:80/http .

# Labels and Annotations Syntax
  # Add a label to all resources created by the app
  acorn run --label key=value .

  # Add a label to resources created for all containers
  acorn run --label containers:key=value .

  # Add a label to the resources created for the volume named "myvolume"
  acorn run --label volumes:myvolume:key=value .

# Link Syntax
  # Link the running acorn application named "mydatabase" into the current app, replacing the container named "db"
  acorn run --link mydatabase:db .

# Secret Syntax
  # Bind the acorn secret named "mycredentials" into the current app, replacing the secret named "creds". See "acorn secrets --help" for more info
  acorn run --secret mycredentials:creds .

# Volume Syntax
  # Create the volume named "mydata" with a size of 5 gigabyes and using the "fast" storage class
  acorn run --volume mydata,size=5G,class=fast .

  # Bind the acorn volume named "mydata" into the current app, replacing the volume named "data", See "acorn volumes --help for more info"
  acorn run --volume mydata:data .

# Automatic upgrades
  # Automatic upgrade for an app will be enabled if '#', '*', or '**' appears in the image's tag. Tags will sorted according to the rules for these special characters described below. The newest tag will be selected for upgrade.

  # '#' denotes a segment of the image tag that should be sorted numerically when finding the newest tag.
  # This example deploys the hello-world app with auto-upgrade enabled and matching all major, minor, and patch versions:
  acorn run myorg/hello-world:v#.#.#

  # '*' denotes a segment of the image tag that should sorted alphabetically when finding the latest tag.
  # In this example, if you had a tag named alpha and a tag named zeta, zeta would be recognized as the newest:
  acorn run myorg/hello-world:*

  # '**' denotes a wildcard. This segment of the image tag won't be considered when sorting. This is useful if your tags have a segment that is unpredictable.
  # This example would sort numerically according to major and minor version (ie v1.2) and ignore anything following the "-":
  acorn run myorg/hello-world:v#.#-**

  # NOTE: Depending on your shell, you may see errors when using '*' and '**'. Using quotes will tell the shell to ignore them so Acorn can parse them:
  acorn run "myorg/hello-world:v#.#-**"

  # Automatic upgrades can be configured explicitly via a flag.
  # In this example, the tag will always be "latest", but acorn will periodically check to see if new content has been pushed to that tag:
  acorn run --auto-upgrade myorg/hello-world:latest

  # To have acorn notify you that an app has an upgrade available and require confirmation before proceeding, set the notify-upgrade flag:
  acorn run --notify-upgrade myorg/hello-world:v#.#.# myapp
  # To proceed with an upgrade you've been notified of:
  acorn update --confirm-upgrade myapp
`})

	// These will produce an error if the flag doesn't exist or a completion function has already been registered for the
	// flag. Not returning the error since neither of these is likely occur.
	if err := cmd.RegisterFlagCompletionFunc("volume", newCompletion(c.ClientFactory, volumeFlagClassCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for -v flag: %v\n", err)
	}
	if err := cmd.RegisterFlagCompletionFunc("compute-class", newCompletion(c.ClientFactory, computeClassFlagCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for --compute-class flag: %v\n", err)
	}
	if err := cmd.RegisterFlagCompletionFunc("region", newCompletion(c.ClientFactory, regionsCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for --region flag: %v\n", err)
	}
	cmd.PersistentFlags().Lookup("dangerous").Hidden = true
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Run struct {
	RunArgs
	Interactive       bool   `usage:"Enable interactive dev mode: build image, stream logs/status in the foreground and stop on exit" short:"i" name:"dev"`
	BidirectionalSync bool   `usage:"In interactive mode download changes in addition to uploading" short:"b"`
	Wait              *bool  `usage:"Wait for app to become ready before command exiting (default true)"`
	Quiet             bool   `usage:"Do not print status" short:"q"`
	Update            bool   `usage:"Update the app if it already exists" short:"u"`
	Replace           bool   `usage:"Replace the app with only defined values, resetting undefined fields to default values" json:"replace,omitempty"` // Replace sets patchMode to false, resulting in a full update, resetting all undefined fields to their defaults
	Image             string `json:"image,omitempty"`

	out    io.Writer
	client ClientFactory
}

type RunArgs struct {
	Name            string   `usage:"Name of app to create" short:"n"`
	Region          string   `usage:"Region in which to deploy the app, immutable"`
	File            string   `short:"f" usage:"Name of the build file" default:"DIRECTORY/Acornfile"`
	Volume          []string `usage:"Bind an existing volume (format existing:vol-name,field=value) (ex: pvc-name:app-data)" short:"v" split:"false"`
	Secret          []string `usage:"Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)" short:"s"`
	Link            []string `usage:"Link external app as a service in the current app (format app-name:container-name)"`
	PublishAll      *bool    `usage:"Publish all (true) or none (false) of the defined ports of application" short:"P"`
	Publish         []string `usage:"Publish port of application (format [public:]private) (ex 81:80)" short:"p"`
	Profile         []string `usage:"Profile to assign default values"`
	Env             []string `usage:"Environment variables to set on running containers" short:"e"`
	Label           []string `usage:"Add labels to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)" short:"l"`
	Annotation      []string `usage:"Add annotations to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)"`
	Dangerous       bool     `usage:"Automatically approve all privileges requested by the application"`
	Output          string   `usage:"Output API request without creating app (json, yaml)" short:"o"`
	TargetNamespace string   `usage:"The name of the namespace to be created and deleted for the application resources"`
	NotifyUpgrade   *bool    `usage:"If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it"`
	AutoUpgrade     *bool    `usage:"Enabled automatic upgrades."`
	Interval        string   `usage:"If configured for auto-upgrade, this is the time interval at which to check for new releases (ex: 1h, 5m)"`
	Memory          []string `usage:"Set memory for a workload in the format of workload=memory. Only specify an amount to set all workloads. (ex foo=512Mi or 512Mi)" short:"m"`
	ComputeClass    []string `usage:"Set computeclass for a workload in the format of workload=computeclass. Specify a single computeclass to set all workloads. (ex foo=example-class or example-class)"`
}

func (s RunArgs) ToOpts() (client.AppRunOptions, error) {
	var (
		opts client.AppRunOptions
		err  error
	)

	opts.Name = s.Name
	opts.Region = s.Region
	opts.Profiles = s.Profile
	opts.TargetNamespace = s.TargetNamespace
	opts.AutoUpgrade = s.AutoUpgrade
	opts.NotifyUpgrade = s.NotifyUpgrade
	opts.AutoUpgradeInterval = s.Interval

	opts.Memory, err = v1.ParseMemory(s.Memory)
	if err != nil {
		return opts, err
	}

	opts.ComputeClasses, err = v1.ParseComputeClass(s.ComputeClass)
	if err != nil {
		return opts, err
	}

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

	opts.Labels, err = v1.ParseScopedLabels(s.Label...)
	if err != nil {
		return opts, err
	}

	opts.Annotations, err = v1.ParseScopedLabels(s.Annotation...)
	if err != nil {
		return opts, err
	}

	opts.Publish, err = v1.ParsePortBindings(s.Publish)
	if err != nil {
		return opts, err
	}

	if s.PublishAll != nil && *s.PublishAll {
		opts.PublishMode = v1.PublishModeAll
	} else if s.PublishAll != nil && !*s.PublishAll {
		opts.PublishMode = v1.PublishModeNone
	}

	return opts, nil
}

// isDirectory checks that the path from the provided directory
// point to a directory. If it does not point to a directory and it points at a file,
// it errors. Otherwise, the function returns false.
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

func buildImage(ctx context.Context, c client.Client, file, cwd string, args []string) (string, error) {
	params, err := build.ParseParams(file, cwd, args)
	if err != nil {
		return "", err
	}

	image, err := c.AcornImageBuild(ctx, file, &client.AcornImageBuildOptions{
		Cwd:  cwd,
		Args: params,
	})
	if err != nil {
		return "", err
	}

	return image.ID, nil
}

func (s *Run) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}
	existingApp, _ := c.AppGet(cmd.Context(), s.Name)
	if existingApp != nil && !s.Update && !s.Replace && !s.Interactive {
		return fmt.Errorf("app \"%s\" already exists", s.Name)
	}

	// Force install prompt if needed
	_, err = c.Info(cmd.Context())
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
	var opts client.AppRunOptions
	// Update mode -> early exits on invalid combinations
	if s.Update && s.Replace {
		return fmt.Errorf("cannot combine --update/-u and --replace/-r")
	}
	if s.Update || s.Replace {
		if s.Interactive {
			return fmt.Errorf("cannot use --update/-u or --replace/-r with --dev/-i")
		}
		//if app does not exist but --update/--replace flag is used create app instead
		if existingApp != nil {
			return updateHelper(cmd, args, s, c, isDir, cwd)
		}
	}

	if s.Interactive {
		d := Dev{
			RunArgs:           s.RunArgs,
			BidirectionalSync: s.BidirectionalSync,
			Quiet:             s.Quiet,
			out:               s.out,
			client:            s.client,
		}
		err := d.Run(cmd, append([]string{}, args...))
		return err
	}

	image := cwd
	if isDir {
		image, err = buildImage(cmd.Context(), c, s.File, cwd, args)
		if err == pflag.ErrHelp {
			return nil
		} else if err != nil {
			return err
		}
	}

	runOpts, err := s.ToOpts()
	if err != nil {
		return err
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

		runOpts.DeployArgs = deployParams
	}

	if s.Output != "" {
		app := client.ToApp(c.GetNamespace(), image, &opts)
		return outputApp(s.out, s.Output, app)
	}

	app, err := rulerequest.PromptRun(cmd.Context(), c, s.Dangerous, image, runOpts)
	if err != nil {
		return err
	}

	fmt.Println(app.Name)

	if s.Wait == nil || *s.Wait {
		return wait.App(cmd.Context(), c, app.Name, s.Quiet)
	}

	return nil
}

func updateHelper(cmd *cobra.Command, args []string, s *Run, c client.Client, isDir bool, cwd string) error {
	var app *apiv1.App
	var err error

	if s.Name != "" {
		app, err = c.AppGet(cmd.Context(), s.Name)
		if err != nil {
			return err
		}
	}
	imageForFlags := s.Image

	if s.Name == "" && !isDir {
		imageForFlags = args[0]
	}

	//updating an already existing app
	if app != nil && imageForFlags == "" {
		imageForFlags = app.Spec.Image
	}
	if _, isPattern := autoupgrade.AutoUpgradePattern(imageForFlags); isPattern {
		imageForFlags = app.Status.AppImage.ID
	}

	//creating an image and updating
	if s.Image == "" && isDir && len(args) > 0 {
		imageForFlags, err = buildImage(cmd.Context(), c, s.File, cwd, args)
		if err == pflag.ErrHelp {
			return nil
		} else if err != nil {
			return err
		}
	}
	//PUT update of a running app
	if s.Replace && !isDir && imageForFlags == "" {
		imageForFlags = s.Image
	}
	_, flags, err := deployargs.ToFlagsFromImage(cmd.Context(), c, imageForFlags)
	if err != nil {
		return err
	}
	deployParams, err := flags.Parse(args)
	if pflag.ErrHelp == err {
		return nil
	} else if err != nil {
		return err
	}

	runOpts, err := s.ToOpts()
	if err != nil {
		return err
	}
	runOpts.DeployArgs = deployParams

	opts := runOpts.ToUpdate()
	opts.Image = imageForFlags

	// Overwrite == true means patchMode == false
	opts.Replace = s.Replace

	if s.Output != "" {
		app, err := client.ToAppUpdate(cmd.Context(), c, s.Name, &opts)
		if err != nil {
			return err
		}
		return outputApp(s.out, s.Output, app)
	}
	if app == nil {
		app, err = rulerequest.PromptRun(cmd.Context(), c, s.Dangerous, imageForFlags, runOpts)
		if err != nil {
			return err
		}
	} else {
		app, err = rulerequest.PromptUpdate(cmd.Context(), c, s.Dangerous, s.Name, opts)
		if err != nil {
			return err
		}
	}
	fmt.Println(app.Name)
	if s.Wait == nil || *s.Wait {
		return wait.App(cmd.Context(), c, app.Name, s.Quiet)
	}
	return nil
}

func outputApp(out io.Writer, format string, app *apiv1.App) error {
	data, err := json.Marshal(app)
	if err != nil {
		return err
	}

	mapData := map[string]any{}
	if err := json.Unmarshal(data, &mapData); err != nil {
		return err
	}

	delete(mapData, "status")
	delete(mapData["metadata"].(map[string]any), "uid")
	delete(mapData["metadata"].(map[string]any), "resourceVersion")
	delete(mapData["metadata"].(map[string]any), "managedFields")
	delete(mapData["metadata"].(map[string]any), "creationTimestamp")

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
