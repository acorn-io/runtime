package cli

import (
	"context"
	"regexp"
	"strings"

	"github.com/acorn-io/baaah/pkg/typed"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/channels"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/project"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

var volumeClassVolumeFlagRegex = regexp.MustCompile("^.*class=([^,]*)$")

type completionFunc func(context.Context, client.Client, string) ([]string, error)

// These define instances when the completion should not happen and the default completion (usually file/directory
// completion should occur). For example, exec should only have one argument, everything after that is completed as
// default by the user's terminal.
type noCompletionOption func([]string) bool

type completion struct {
	client           ClientFactory
	completionFunc   completionFunc
	successDirective cobra.ShellCompDirective

	noCompletionOptions []noCompletionOption
}

func removeExistingArgs(result, args []string) []string {
	for i := 0; i < len(result); {
		if slices.Contains(args, result[i]) {
			result = append(result[:i], result[i+1:]...)
		} else {
			i++
		}
	}

	return result
}

func onlyNumArgs(n int) noCompletionOption {
	return func(args []string) bool {
		return len(args) >= n
	}
}

func newCompletion(c ClientFactory, cf completionFunc) *completion {
	return &completion{
		client:           c,
		completionFunc:   cf,
		successDirective: cobra.ShellCompDirectiveNoFileComp,
	}
}

func (a *completion) withShouldCompleteOptions(opts ...noCompletionOption) *completion {
	a.noCompletionOptions = append(a.noCompletionOptions, opts...)
	return a
}

func (a *completion) withSuccessDirective(d cobra.ShellCompDirective) *completion {
	a.successDirective = d
	return a
}

func (a *completion) complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	for _, o := range a.noCompletionOptions {
		if o(args) {
			return nil, cobra.ShellCompDirectiveDefault
		}
	}
	c, err := a.client.CreateDefault()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	result, err := a.completionFunc(cmd.Context(), c, toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return removeExistingArgs(result, args), a.successDirective
}

func appsThenSecretsCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	result, err := appsCompletion(ctx, c, toComplete)
	if err != nil || len(result) > 0 {
		return result, err
	}
	return secretsCompletion(ctx, c, toComplete)
}

func appsThenContainersCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	// If the toComplete has a '.', then the user is looking for a container.
	if strings.Contains(toComplete, ".") {
		return containersCompletion(ctx, c, toComplete)
	}

	return appsCompletion(ctx, c, toComplete)
}

func appsCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	var result []string
	apps, err := c.AppList(ctx)
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		if strings.HasPrefix(app.Name, toComplete) {
			result = append(result, app.Name)
		}
	}

	return result, nil
}

func containersCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	var result []string
	containers, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if strings.HasPrefix(container.Name, toComplete) {
			result = append(result, container.Name)
		}
	}

	return result, nil
}

func jobsCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	var result []string
	jobs, err := c.JobList(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, job := range jobs {
		if strings.HasPrefix(job.Name, toComplete) {
			result = append(result, job.Name)
		}
	}

	return result, nil
}

// acornContainerCompletion will complete the `-c` flag for various commands like exec. It must look at all apps and
// then for all containers on status.appSpec.Containers to produce a list of possibilities.
func acornContainerCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	apps, err := c.AppList(ctx)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, app := range apps {
		for _, entry := range typed.Sorted(app.Status.AppSpec.Containers) {
			if strings.HasPrefix(entry.Key, toComplete) {
				result = append(result, entry.Key)
			}
		}
	}

	return result, nil
}

// onlyAppsWithAcornContainer will look for completions for apps and pods containers when the container name is
// empty. If the container name is not empty, then it will only look for apps that have the specified container name in
// their appSpec.
func onlyAppsWithAcornContainer(containerName string) completionFunc {
	return func(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
		// If no container is specified, then look for apps and pod containers.
		if containerName == "" {
			return appsThenContainersCompletion(ctx, c, toComplete)
		}

		// If a container has been specified, then only produce completions of apps that have such a container.
		apps, err := c.AppList(ctx)
		if err != nil {
			return nil, err
		}

		var result []string
		for _, app := range apps {
			for container := range app.Status.AppSpec.Containers {
				if container == containerName && strings.HasPrefix(app.Name, toComplete) {
					result = append(result, app.Name)
					break
				}
			}
		}

		return result, nil
	}
}

func imagesCompletion(allowDigest bool) completionFunc {
	return func(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
		images, err := c.ImageList(ctx)
		if err != nil {
			return nil, err
		}

		var result []string
		var tagMatched bool
		for _, image := range images {
			tagMatched = false
			for _, tag := range image.Tags {
				if strings.HasPrefix(tag, toComplete) {
					result = append(result, tag)
					tagMatched = true
				}
			}

			// Don't include the digest if a tag matched.
			if allowDigest && !tagMatched {
				digest := strings.TrimPrefix(image.Digest, "sha256:")[:12]
				if strings.HasPrefix(digest, toComplete) {
					result = append(result, digest)
				}
			}
		}

		return result, nil
	}
}

func credentialsCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	credentials, err := c.CredentialList(ctx)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, credential := range credentials {
		if strings.HasPrefix(credential.Name, toComplete) {
			result = append(result, credential.Name)
		}
	}

	return result, nil
}

func volumesCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	volumes, err := c.VolumeList(ctx)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, volume := range volumes {
		if strings.HasPrefix(volume.Name, toComplete) {
			result = append(result, volume.Name)
		}
	}

	return result, nil
}

func secretsCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	secrets, err := c.SecretList(ctx)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, secret := range secrets {
		if strings.HasPrefix(secret.Name, toComplete) {
			result = append(result, secret.Name)
		}
	}

	return result, nil
}

func projectsCompletion(f ClientFactory) completionFunc {
	return func(ctx context.Context, _ client.Client, toComplete string) ([]string, error) {
		var acornConfigFile string
		if f != nil {
			acornConfigFile = f.AcornConfigFile()
		}

		projects, _, err := project.List(ctx, false, project.Options{
			AcornConfigFile: acornConfigFile,
		})
		if err != nil {
			return nil, err
		}

		var result []string
		for _, project := range projects {
			if strings.HasPrefix(project, toComplete) {
				result = append(result, project)
			}
		}

		return result, nil
	}
}

func volumeClassCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	volumeClasses, err := c.VolumeClassList(ctx)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, volumeClass := range volumeClasses {
		if strings.HasPrefix(volumeClass.Name, toComplete) {
			result = append(result, volumeClass.Name)
		}
	}

	return result, nil
}

func computeClassCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	computeClasses, err := c.ComputeClassList(ctx)
	if err != nil {
		return nil, err
	}

	var result []string

	for _, volumeClass := range computeClasses {
		if strings.HasPrefix(volumeClass.Name, toComplete) {
			result = append(result, volumeClass.Name)
		}
	}

	return result, nil
}

func volumeFlagClassCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	matches := volumeClassVolumeFlagRegex.FindAllStringSubmatch(toComplete, 1)
	if len(matches) == 0 {
		return nil, nil
	}

	// If the regexp matches, then this is a flag completion of the form `-v foo:size=5G,class=..` and toComplete should be what follows `class=`
	actualToComplete := matches[0][1]
	result, err := volumeClassCompletion(ctx, c, actualToComplete)

	// Trim the actualToComplete from the end so that we can append below to get the full completion.
	toComplete = strings.TrimSuffix(toComplete, actualToComplete)

	// Add the rest of the toComplete flag to the completion.
	for i := range result {
		result[i] = toComplete + result[i]
	}

	return result, err
}

func computeClassFlagCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	var (
		computeClassFlagCompletion = regexp.MustCompile("^.*[,|=]([^,]*)$")
		actualToComplete           = toComplete
	)

	if matches := computeClassFlagCompletion.FindAllStringSubmatch(toComplete, 1); matches != nil {
		actualToComplete = matches[0][1]
	}

	result, err := computeClassCompletion(ctx, c, actualToComplete)

	// Trim the actualToComplete from the end so that we can append below to get the full completion.
	toComplete = strings.TrimSuffix(toComplete, actualToComplete)

	// Add the rest of the toComplete flag to the completion.
	for i := range result {
		result[i] = toComplete + result[i]
	}

	return result, err
}

func regionsCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	regions, err := c.RegionList(ctx)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, region := range regions {
		if strings.HasPrefix(region.Name, toComplete) {
			result = append(result, region.Name)
		}
	}

	return result, nil
}

func eventsCompletion(ctx context.Context, c client.Client, toComplete string) ([]string, error) {
	var result []string
	events, err := c.EventStream(ctx, &client.EventStreamOptions{})
	if err != nil {
		return nil, err
	}

	matched := make(map[string]struct{})
	if err := channels.ForEach(ctx, events, func(e apiv1.Event) error {
		completions := []string{
			// Name prefix completion
			e.Name,
		}

		if e.Resource != nil {
			// Resource prefix completions
			completions = append(completions,
				e.Resource.String(),
				e.Resource.Kind,
			)
		}

		for _, completion := range completions {
			if _, ok := matched[completion]; ok {
				// Completion already added to results
				return nil
			}

			if strings.HasPrefix(completion, toComplete) {
				result = append(result, completion)
				matched[completion] = struct{}{}
			}
		}

		return nil
	}); !channels.NilOrCanceled(err) {
		return nil, err
	}

	return result, nil
}
