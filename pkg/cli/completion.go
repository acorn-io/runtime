package cli

import (
	"context"
	"strings"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

type completionFunc func(context.Context, client.Client, string) ([]string, error)

// These define instances when the completion should not happen and the default completion (usually file/directory
// completion should occur). For example, exec should only have one argument, everything after that is completed as
// default by the user's terminal.
type noCompletionOption func([]string) bool

type completion struct {
	client           client.ClientFactory
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

func newCompletion(c client.ClientFactory, cf completionFunc) *completion {
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
	var opts *client.ContainerReplicaListOptions
	if strings.Contains(toComplete, ".") {
		opts = &client.ContainerReplicaListOptions{App: strings.Split(toComplete, ".")[0]}
	}

	containers, err := c.ContainerReplicaList(ctx, opts)
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
