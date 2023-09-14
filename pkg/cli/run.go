package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/dev"
	"github.com/acorn-io/runtime/pkg/imageallowrules"
	"github.com/acorn-io/runtime/pkg/imagesource"
	"github.com/acorn-io/runtime/pkg/rulerequest"
	"github.com/acorn-io/runtime/pkg/wait"
	"github.com/acorn-io/z"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/yaml"
)

func NewRun(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Run{out: c.StdOut, client: c.ClientFactory}, cobra.Command{
		Use:               "run [flags] IMAGE|DIRECTORY [acorn args]",
		SilenceUsage:      true,
		Short:             "Run an app from an image or Acornfile",
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).withSuccessDirective(cobra.ShellCompDirectiveDefault).withShouldCompleteOptions(onlyNumArgs(1)).complete,
		Example: `
 # Build and run from a directory
   acorn run .

 # Run from an image
   acorn run ghcr.io/acorn-io/hello-world

 # Automatic upgrades
   # Automatic upgrade for an app will be enabled if '#', '*', or '**' appears in the image's tag. Tags will be sorted according to the rules for these special characters described below. The newest tag will be selected for upgrade.
   
   # '#' denotes a segment of the image tag that should be sorted numerically when finding the newest tag.

   # This example deploys the hello-world app with auto-upgrade enabled and matching all major, minor, and patch versions:
   acorn run myorg/hello-world:v#.#.#

   # '*' denotes a segment of the image tag that should sorted alphabetically when finding the latest tag.
  
   # In this example, if you had a tag named alpha and a tag named zeta, zeta would be recognized as the newest:
   acorn run myorg/hello-world:*

   # '**' denotes a wildcard. This segment of the image tag won't be considered when sorting. This is useful if your tags have a segment that is unpredictable.
   
   # This example would sort numerically according to major and minor version (i.e. v1.2) and ignore anything following the "-":
   acorn run myorg/hello-world:v#.#-**

   # NOTE: Depending on your shell, you may see errors when using '*' and '**'. Using quotes will tell the shell to ignore them so acorn can parse them:
   acorn run "myorg/hello-world:v#.#-**"

   # Automatic upgrades can be configured explicitly via a flag.

   # In this example, the tag will always be "latest", but acorn will periodically check to see if new content has been pushed to that tag:
   acorn run --auto-upgrade myorg/hello-world:latest

   # To have acorn notify you that an app has an upgrade available and require confirmation before proceeding, set the notify-upgrade flag:
   acorn run --notify-upgrade myorg/hello-world:v#.#.# myapp

   # To proceed with an upgrade you've been notified of:
   acorn update --confirm-upgrade myapp`,
	})

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
	cmd.Flags().SetInterspersed(false)
	toggleHiddenFlags(cmd, hideRunFlags, true)

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.Short + "\n")
		fmt.Println(cmd.UsageString())
	})
	return cmd
}

const AdvancedHelp = `    
More Usages:
     # Publish and Expose Port Syntax
     - Publish port 80 for any containers that define it as a port
      	acorn run -p 80 .

     - Publish container "myapp" using the hostname app.example.com
      	acorn run --publish app.example.com:myapp .

     - Expose port 80 to the rest of the cluster as port 8080
      	acorn run --expose 8080:80/http .

     # Labels and Annotations Syntax
     - Add a label to all resources created by the app
       	acorn run --label key=value .

     - Add a label to resources created for all containers
      	acorn run --label containers:key=value .

     - Add a label to the resources created for the volume named "myvolume"
      	acorn run --label volumes:myvolume:key=value .

     # Link Syntax
     - Link the running acorn application named "mydatabase" into the current app, replacing the container named "db"
       	acorn run --link mydatabase:db .

     # Secret Syntax
     - Bind the acorn secret named "mycredentials" into the current app, replacing the secret named "creds". See "acorn secrets --help" for more info
         acorn run --secret mycredentials:creds .

     # Volume Syntax
     - Create the volume named "mydata" with a size of 5 gigabyes and using the "fast" storage class
        acorn run --volume mydata,size=5G,class=fast .
     - Bind the acorn volume named "mydata" into the current app, replacing the volume named "data", See "acorn volumes --help for more info"
        acorn run --volume mydata:data .`

var hideRunFlags = []string{"dangerous", "memory", "secret", "volume", "region", "publish-all",
	"publish", "link", "label", "interval", "env", "compute-class", "annotation", "update", "replace"}

type Run struct {
	RunArgs
	Dev               bool  `usage:"Enable interactive dev mode: build image, stream logs/status in the foreground and stop on exit" short:"i"`
	BidirectionalSync bool  `usage:"In interactive mode download changes in addition to uploading" short:"b"`
	Wait              *bool `usage:"Wait for app to become ready before command exiting (default: true)"`
	Quiet             bool  `usage:"Do not print status" short:"q"`
	Update            bool  `usage:"Update the app if it already exists" short:"u"`
	Replace           bool  `usage:"Replace the app with only defined values, resetting undefined fields to default values" json:"replace,omitempty"` // Replace sets patchMode to false, resulting in a full update, resetting all undefined fields to their defaults
	HelpAdvanced      bool  `usage:"Show verbose help text"`

	out    io.Writer
	client ClientFactory
}

type RunArgs struct {
	UpdateArgs
	Name string `usage:"Name of app to create" short:"n"`
}

func (s RunArgs) ToOpts() (client.AppRunOptions, error) {
	var (
		opts client.AppRunOptions
		err  error
	)

	opts.Name = s.Name
	opts.Region = s.Region
	opts.Profiles = s.Profile
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

func (s *Run) Run(cmd *cobra.Command, args []string) (err error) {
	if s.HelpAdvanced {
		setAdvancedHelp(cmd, hideRunFlags, AdvancedHelp)
		return cmd.Help()
	}
	defer func() {
		if errors.Is(err, pflag.ErrHelp) {
			err = nil
		}
	}()

	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	var (
		imageSource = imagesource.NewImageSource(s.client.AcornConfigFile(), s.File, args, s.Profile, nil, z.Dereference(s.AutoUpgrade))
		app         *apiv1.App
		updated     bool
	)

	opts, err := s.ToOpts()
	if err != nil {
		return err
	}

	// If auto-upgrade is not set, set it to true if auto-upgrade is implied
	if !z.Dereference(opts.AutoUpgrade) && autoupgrade.Implied(imageSource.Image, s.Interval, z.Dereference(opts.NotifyUpgrade)) {
		opts.AutoUpgrade = z.Pointer(true)
	}

	// Force install prompt if needed
	_, err = c.Info(cmd.Context())
	if err != nil {
		return err
	}

	if s.Dev {
		return dev.Dev(cmd.Context(), c, &dev.Options{
			ImageSource:       imageSource,
			Run:               opts,
			Replace:           s.Replace,
			Dangerous:         s.Dangerous,
			BidirectionalSync: s.BidirectionalSync,
		})
	}

	defer func() {
		if err == nil && (s.Wait == nil || *s.Wait) && app != nil {
			app, getErr := c.AppGet(cmd.Context(), app.Name)
			if getErr == nil && (app.GetStopped() || !app.DeletionTimestamp.IsZero()) {
				return
			}
			_ = wait.App(cmd.Context(), c, app.Name, s.Quiet)
		}
	}()

	if s.Replace || s.Update {
		if s.Output != "" {
			return fmt.Errorf("--output can not be combined with --update or --replace")
		}
		app, updated, err = s.update(cmd.Context(), c, imageSource, opts)
		if err != nil {
			return err
		}
		if updated {
			fmt.Println(app.Name)
			return nil
		}
	}

	image, deployArgs, err := imageSource.GetImageAndDeployArgs(cmd.Context(), c)
	if err != nil {
		if apierrors.IsUnauthorized(err) {
			logrus.Debugf("Error pulling %s: %v", image, err)
			return fmt.Errorf("not authorized to pull %s - Use `acorn login <REGISTRY>` to login to the registry", image)
		}
		err = client.TranslateNotAllowed(err)
		if naErr := (*imageallowrules.ErrImageNotAllowed)(nil); errors.As(err, &naErr) {
			if _, isPattern := autoupgrade.AutoUpgradePattern(image); isPattern {
				err.(*imageallowrules.ErrImageNotAllowed).Image = image
				logrus.Debugf("Valid tags for pattern %s were not allowed to run: %v", image, naErr)
				if choice, promptErr := rulerequest.HandleNotAllowed(s.Dangerous, image); promptErr != nil {
					return promptErr
				} else if choice != "NO" {
					iarErr := rulerequest.CreateImageAllowRule(cmd.Context(), c, image, choice)
					if iarErr != nil {
						return iarErr
					}
					return s.Run(cmd, args)
				}
			}
		}
		return err
	}
	opts.DeployArgs = deployArgs

	if s.Output != "" {
		app := client.ToApp(c.GetNamespace(), image, &opts)
		return outputApp(s.out, s.Output, app)
	}

	app, err = rulerequest.PromptRun(cmd.Context(), c, s.Dangerous, image, opts)
	if err != nil {
		return err
	}
	fmt.Println(app.Name)
	return nil
}

func (s *Run) update(ctx context.Context, c client.Client, imageSource imagesource.ImageSource, opts client.AppRunOptions) (*apiv1.App, bool, error) {
	if s.Name == "" {
		return nil, false, fmt.Errorf("--name is required for --update or --replace")
	}

	app, err := c.AppGet(ctx, s.Name)
	if apierrors.IsNotFound(err) {
		if !imageSource.IsImageSet() {
			return nil, false, fmt.Errorf("acorn \"%s\" is missing but can not be created without specifying an image to run or build", s.Name)
		}
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	updateOpts := opts.ToUpdate()
	updateOpts.Replace = s.Replace

	if imageSource.IsImageSet() {
		image, deployArgs, err := imageSource.GetImageAndDeployArgs(ctx, c)
		if err != nil {
			return nil, false, err
		}
		updateOpts.Image = image
		updateOpts.DeployArgs = deployArgs
	} else if len(imageSource.Args) > 0 {
		imageSource.Image = app.Status.AppImage.Name
		if _, updateOpts.DeployArgs, err = imageSource.GetImageAndDeployArgs(ctx, c); err != nil {
			return nil, false, err
		}
	}

	app, err = rulerequest.PromptUpdate(ctx, c, s.Dangerous, app.Name, updateOpts)
	if err != nil {
		return nil, false, err
	}

	return app, true, nil
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

func toggleHiddenFlags(cmd *cobra.Command, flagsToHide []string, hide bool) {
	for _, flag := range flagsToHide {
		cmd.PersistentFlags().Lookup(flag).Hidden = hide
	}
}

func setAdvancedHelp(cmd *cobra.Command, hideRunFlags []string, advancedHelp string) {
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.Short + "\n")
		// toggle advanced flags on before printing flags out in cmd.UsageString
		toggleHiddenFlags(cmd, hideRunFlags, false)
		fmt.Println(cmd.UsageString())
		fmt.Println(advancedHelp)
	})
}
