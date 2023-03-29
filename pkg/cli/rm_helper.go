package cli

import (
	"fmt"
	"strings"

	"github.com/acorn-io/acorn/pkg/prompt"
	"github.com/pterm/pterm"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func addRmObject(rmObjects *RmObjects, obj string) {
	switch strings.ToLower(obj) {
	case "app":
		rmObjects.App = true
	case "container":
		rmObjects.Container = true
	case "secret":
		rmObjects.Secret = true
	case "volume":
		rmObjects.Volume = true
	case "a":
		rmObjects.App = true
	case "c":
		rmObjects.Container = true
	case "s":
		rmObjects.Secret = true
	case "v":
		rmObjects.Volume = true
	default:
		pterm.Warning.Printf("%s is not a valid type\n", obj)
	}
}

func getSecretsToRemove(arg string, client client.Client, cmd *cobra.Command) ([]string, error) {
	var result []string
	secrets, err := client.SecretList(cmd.Context())
	apps, _ := client.AppList(cmd.Context())

	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		aliasList := aliases(&secret, apps)
		if len(aliasList) != 0 {
			secretName := strings.Split(aliasList[0], ".")
			if len(secretName) != 0 && arg == secretName[0] {
				result = append(result, secret.Name)
			}
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
		if arg == volume.Status.AppName {
			result = append(result, volume.Name)
		}
	}
	return result, nil
}
func getContainersToDelete(arg string, client client.Client, cmd *cobra.Command) ([]string, error) {
	var result []string
	containers, err := client.ContainerReplicaList(cmd.Context(), nil)
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if arg == strings.Split(container.Name, ".")[0] {
			result = append(result, container.Name)
		}
	}
	return result, nil
}
func removeContainer(arg string, c client.Client, cmd *cobra.Command, force bool) error {
	conToDel, err := getContainersToDelete(arg, c, cmd)
	if len(conToDel) == 0 {
		pterm.Info.Println("No containers associated with " + arg)
		return nil
	}
	if !force {
		for _, con := range conToDel {
			pterm.FgRed.Println(con)
		}
		err := prompt.Remove("containers")
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	for _, con := range conToDel {
		_, err := c.ContainerReplicaDelete(cmd.Context(), con)
		if err != nil {
			return fmt.Errorf("deleting container %s: %w", con, err)
		}

		fmt.Println("Removed: " + con)
		continue
	}
	return nil
}
func removeVolume(arg string, c client.Client, cmd *cobra.Command, force bool) error {
	volToDel, err := getVolumesToDelete(arg, c, cmd)
	if err != nil {
		return err
	}
	if len(volToDel) == 0 {
		pterm.Info.Println("No volumes associated with " + arg)
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
			fmt.Println("Removed: " + vol)
			continue
		} else {
			fmt.Printf("Error: No such volume: %s\n", vol)
		}
	}
	return nil
}
func removeApp(arg string, c client.Client, cmd *cobra.Command, force bool) error {
	if !force {
		pterm.FgRed.Println(arg)
		err := prompt.Remove("app")
		if err != nil {
			return err
		}
	}
	app, err := c.AppDelete(cmd.Context(), arg)

	if err != nil {
		return fmt.Errorf("deleting app %s: %w", arg, err)
	}
	if app != nil {
		fmt.Println("Removed: " + arg)
	} else {
		fmt.Printf("Error: No such app: %s\n", arg)
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
