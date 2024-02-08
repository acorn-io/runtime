package cli

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/install"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

func NewInstall(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Install{client: c.ClientFactory}, cobra.Command{
		Use: "install [flags]",
		Example: `
acorn install`,
		Aliases:      []string{"init"},
		SilenceUsage: true,
		Short:        "Install and configure acorn in the cluster",
		Args:         cobra.NoArgs,
	})
	cmd.PersistentFlags().Lookup("dev").Hidden = true
	return cmd
}

type Install struct {
	SkipChecks bool   `usage:"Bypass installation checks"`
	Quiet      bool   `usage:"Only output errors encountered during installation"`
	Image      string `usage:"Override the default image used for the deployment"`
	Output     string `usage:"Output manifests instead of applying them (json, yaml)" short:"o"`

	APIServerReplicas                  *int     `usage:"acorn-api deployment replica count" name:"api-server-replicas"`
	APIServerPodAnnotations            []string `usage:"annotations to apply to acorn-api pods" name:"api-server-pod-annotations" split:"false"`
	ControllerReplicas                 *int     `usage:"acorn-controller deployment replica count"`
	ControllerServiceAccountAnnotation []string `usage:"annotation to apply to the acorn-system service account"`

	Dev string `usage:"Development overlay install"`

	apiv1.Config
	client ClientFactory
}

func (i *Install) dev(ctx context.Context, imageName string, opts *install.Options) error {
	c, err := i.client.CreateDefault()
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(imageName)
	if err != nil {
		return err
	}

	cfg, err := i.client.Options().CLIConfig()
	if err != nil {
		return err
	}

	creds, err := credentials.NewStore(cfg, c)
	if err != nil {
		return err
	}

	auth, ok, err := creds.Get(ref.Context().RegistryStr())
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("credential not found for %s, run acorn/docker login first", imageName)
	}

	return install.Dev(ctx, imageName, auth, opts)
}

func (i *Install) Run(cmd *cobra.Command, _ []string) error {
	var image = system.DefaultImage()
	if i.Image != "" {
		image = i.Image
	}

	controllerSAAnnotations, err := parseAnnotations(i.ControllerServiceAccountAnnotation)
	if err != nil {
		return fmt.Errorf("invalid --controller-service-account-annotation %w", err)
	}

	apiPodAnnotations, err := parseAnnotations(i.APIServerPodAnnotations)
	if err != nil {
		return fmt.Errorf("invalid --api-server-pod-annotations: %w", err)
	}

	opts := &install.Options{
		SkipChecks:                          i.SkipChecks,
		Quiet:                               i.Quiet,
		OutputFormat:                        i.Output,
		Config:                              i.Config,
		APIServerReplicas:                   i.APIServerReplicas,
		APIServerPodAnnotations:             apiPodAnnotations,
		ControllerReplicas:                  i.ControllerReplicas,
		ControllerServiceAccountAnnotations: controllerSAAnnotations,
	}

	if i.Dev != "" {
		return i.dev(cmd.Context(), i.Dev, opts)
	}

	return install.Install(cmd.Context(), image, opts)
}

func parseAnnotations(annotations []string) (map[string]string, error) {
	result := make(map[string]string, len(annotations))
	for _, anno := range annotations {
		k, v, ok := strings.Cut(anno, "=")
		if !ok {
			return nil, fmt.Errorf("annotation must be in key=value format got [%s]", anno)
		}
		result[k] = v
	}
	return result, nil
}
