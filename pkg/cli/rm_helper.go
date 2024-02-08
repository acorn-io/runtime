package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/prompt"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8swait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/acorn-io/runtime/pkg/client"
	"github.com/spf13/cobra"
)

func getSecretsToRemove(arg string, client client.Client, cmd *cobra.Command) ([]string, error) {
	var result []string
	secrets, err := client.SecretList(cmd.Context())
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		// We only want to delete secrets for the given app and not nested apps
		if after, found := strings.CutPrefix(secret.Name, arg+"."); found && !strings.ContainsRune(after, '.') {
			result = append(result, secret.Name)
		}
	}
	return result, nil
}

func getVolumesToDelete(arg string, client client.Client, cmd *cobra.Command) ([]string, error) {
	var result []string
	volumes, err := client.VolumeList(cmd.Context())
	if err != nil {
		return nil, err
	}

	for _, volume := range volumes {
		if arg == volume.Status.AppName { // if the volume is a part of the app
			result = append(result, volume.Name)
		}
	}
	return result, nil
}

func removeVolume(arg string, c client.Client, cmd *cobra.Command, force bool) error {
	volToDel, err := getVolumesToDelete(arg, c, cmd)
	if err != nil {
		return err
	}
	if len(volToDel) == 0 {
		return nil
	}
	if !force {
		for _, vol := range volToDel {
			pterm.FgRed.Println(vol)
		}
		err = prompt.Remove("volumes")
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	for _, vol := range volToDel {
		v, err := c.VolumeDelete(cmd.Context(), vol)
		if err != nil {
			return fmt.Errorf("deleting volume %s: %w", arg, err)
		}
		if v != nil {
			fmt.Println("Removed volume: " + vol)
			continue
		}

		fmt.Printf("Error: No such volume: %s\n", vol)
	}
	return nil
}

func deleteOthers(ctx context.Context, c client.Client, arg string) (bool, error) {
	_, err := c.ContainerReplicaGet(ctx, arg)
	if err == nil {
		_, err = c.ContainerReplicaDelete(ctx, arg)
		if err == nil {
			fmt.Println("Removed container: " + arg)
		}
		return true, err
	} else if kclient.IgnoreNotFound(err) != nil {
		return false, err
	}

	_, err = c.VolumeGet(ctx, arg)
	if err == nil {
		_, err = c.VolumeDelete(ctx, arg)
		if err == nil {
			fmt.Println("Removed volume: " + arg)
		}
		return true, err
	}

	if kclient.IgnoreNotFound(err) != nil {
		return false, err
	}

	_, err = c.SecretGet(ctx, arg)
	if err == nil {
		_, err = c.SecretDelete(ctx, arg)
		if err == nil {
			fmt.Println("Removed secret: " + arg)
		}
		return true, err
	} else if kclient.IgnoreNotFound(err) != nil {
		return false, err
	}

	return false, nil
}

func removeAcorn(ctx context.Context, c client.Client, arg string, ignoreCleanup, wait bool) error {
	var app *v1.App
	err := retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		app, err = c.AppDelete(ctx, arg)
		return
	})
	if err != nil {
		return fmt.Errorf("deleting app %s: %w", arg, err)
	}

	if app == nil {
		if strings.Contains(arg, ".") {
			if ok, err := deleteOthers(ctx, c, arg); err != nil {
				return err
			} else if ok {
				return nil
			}
		}
		fmt.Printf("Error: No such app: %s\n", arg)
		return nil
	}

	if ignoreCleanup {
		// There are situations where an app being deleted the first time with the --ignore-cleanup flag will fail at this
		// step because the server thinks that the app is not being deleted. Retrying here will work around this issue.
		if err = retry.OnError(k8swait.Backoff{
			Steps:    5,
			Duration: 500 * time.Millisecond,
			Factor:   2,
			Jitter:   0.1,
		}, func(err error) bool {
			var statusErr *apierrors.StatusError
			return errors.As(err, &statusErr) && statusErr.Status().Code == http.StatusBadRequest && strings.HasSuffix(statusErr.Status().Message, "it is not being deleted")
		}, func() error {
			return c.AppIgnoreDeleteCleanup(ctx, arg)
		}); err != nil {
			return fmt.Errorf("skipping cleanup for app %s: %w", arg, err)
		}
	}

	// Ensure this gets printed whether we wait or not
	defer func() {
		if err == nil || apierrors.IsNotFound(err) {
			fmt.Println("Removed: " + arg)
		}
	}()

	if wait {
		for {
			if _, err = c.AppGet(ctx, arg); apierrors.IsNotFound(err) {
				return nil
			} else if err != nil {
				logrus.Debugf("Error getting app for removal check: %v", err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
			fmt.Printf("Waiting for app %s to be removed...\n", arg)
		}
	}

	return nil
}

func removeSecret(arg string, c client.Client, cmd *cobra.Command, force bool) error {
	secToDel, err := getSecretsToRemove(arg, c, cmd)
	if len(secToDel) == 0 {
		pterm.Info.Println("No secrets associated with " + arg)
		return nil
	}
	if !force {
		for _, sec := range secToDel {
			pterm.FgRed.Println(sec)
		}
		err = prompt.Remove("secrets")
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	for _, sec := range secToDel {
		secret, err := c.SecretDelete(cmd.Context(), sec)
		if err != nil {
			return fmt.Errorf("deleting secret %s: %w", sec, err)
		}
		if secret != nil {
			fmt.Println("Removed: " + sec)
			continue
		}
	}
	return nil
}
