package login

import (
	"context"
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/typed"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/prompt"
	"github.com/charmbracelet/glamour"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func Secrets(ctx context.Context, c client.Client, app *apiv1.App) (*apiv1.App, error) {
	for secretName, secret := range app.Status.AppSpec.Secrets {
		if strings.HasPrefix(secret.Type, v1.SecretTypeCredentialPrefix) {
			updatedApp, err := loginSecret(ctx, c, app, secretName)
			if err != nil {
				return nil, err
			}
			if updatedApp != nil {
				app = updatedApp
			}
		}
	}

	for _, acorn := range app.Status.AppStatus.Acorns {
		if acorn.AcornName != "" {
			if err := loginApp(ctx, c, acorn.AcornName); err != nil {
				return nil, err
			}
		}
	}

	for _, service := range app.Status.AppStatus.Services {
		if service.ServiceAcornName != "" {
			if err := loginApp(ctx, c, service.ServiceAcornName); err != nil {
				return nil, err
			}
		}
	}

	return app, nil
}

func loginApp(ctx context.Context, c client.Client, appName string) error {
	app, err := c.AppGet(ctx, appName)
	if apierror.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	_, err = Secrets(ctx, c, app)
	return err
}

func secretIsOk(app *apiv1.App, secretName string) (string, bool) {
	return app.Status.AppStatus.Secrets[secretName].LinkOverride,
		!app.Status.AppStatus.Secrets[secretName].LoginRequired &&
			app.Status.AppStatus.Secrets[secretName].LinkOverride != ""
}

func printInstructions(app *apiv1.App, secretName string) error {
	secretDisplayName := app.Name + "." + secretName
	instructions := app.Status.AppStatus.Secrets[secretName].LoginInstructions
	if instructions == "" {
		return nil
	}

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return err
	}

	msg, err := r.Render(instructions)
	if err != nil {
		return err
	}

	fmt.Printf("Please follow the instructions below to login to [%s]:\n", secretDisplayName)
	fmt.Print(msg)
	fmt.Println()
	return nil
}

func bindSecret(ctx context.Context, c client.Client, app *apiv1.App, targetSecretName, overrideSecretName string) (*apiv1.App, error) {
	parts := append(strings.Split(app.Name, "."), targetSecretName)
	appName := parts[0]
	targetSecretName = strings.Join(parts[1:], ".")

	return c.AppUpdate(ctx, appName, &client.AppUpdateOptions{
		Secrets: []v1.SecretBinding{
			{
				Secret: overrideSecretName,
				Target: targetSecretName,
			},
		},
	})
}

func createSecret(ctx context.Context, c client.Client, app *apiv1.App, secretName string) (*apiv1.App, error) {
	secretType := app.Status.AppSpec.Secrets[secretName].Type

	if err := printInstructions(app, secretName); err != nil {
		return nil, err
	}

	asked := map[string]struct{}{}
	data := map[string][]byte{}
	paramsData := app.Status.AppSpec.Secrets[secretName].Params.GetData()
	if promptOrder, exists := paramsData["promptOrder"].([]interface{}); exists {
		for _, k := range promptOrder {
			if key, ok := k.(string); ok {
				if def, ok := app.Status.AppSpec.Secrets[secretName].Data[key]; ok {
					message := key
					if def != "" {
						message += fmt.Sprintf(" (default: %s)", def)
					}
					value, err := prompt.Password(message)
					if err != nil {
						return nil, err
					}
					if len(value) == 0 {
						value = []byte(def)
					}
					data[key] = value
					asked[key] = struct{}{}
				}
			}
		}
	}
	for _, key := range typed.SortedKeys(app.Status.AppSpec.Secrets[secretName].Data) {
		if _, asked := asked[key]; asked {
			continue
		}
		def := app.Status.AppSpec.Secrets[secretName].Data[key]
		message := key
		if def != "" {
			message += fmt.Sprintf(" (default: %s)", def)
		}
		value, err := prompt.Password(message)
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			value = []byte(def)
		}
		data[key] = value
	}

	secret, err := c.SecretCreate(ctx, secretName+"-", secretType, data)
	if err != nil {
		return nil, err
	}

	return bindSecret(ctx, c, app, secretName, secret.Name)
}

func loginSecret(ctx context.Context, c client.Client, app *apiv1.App, secretName string) (*apiv1.App, error) {
	secretType := app.Status.AppSpec.Secrets[secretName].Type
	secretDisplayName := app.Name + "." + secretName

	if existing, ok := secretIsOk(app, secretName); ok {
		change, err := prompt.Bool(fmt.Sprintf("Credential [%s] is configured to [%s], do you want to change it",
			secretDisplayName, existing), false)
		if err != nil || !change {
			return nil, err
		}
		fmt.Println()
	}

	secrets, err := c.SecretList(ctx)
	if err != nil {
		return nil, err
	}

	var (
		secretChoiceName []string
		displayText      []string
	)
	for _, secret := range secrets {
		if secret.Type == secretType {
			secretChoiceName = append(secretChoiceName, secret.Name)
			displayText = append(displayText, "Existing: "+secret.Name+fmt.Sprintf(" (Keys: [%s], Created: [%s])",
				strings.Join(secret.Keys, ", "), secret.CreationTimestamp))
		}
	}

	if len(secretChoiceName) > 0 {
		def := "Enter a new credential"
		choice, err := prompt.Choice(fmt.Sprintf("Choose an existing credential or enter a new one for [%s]",
			secretDisplayName), append(displayText, def), def)
		if err != nil {
			return nil, err
		}
		if choice == def {
			return createSecret(ctx, c, app, secretName)
		}
		for i, displayTest := range displayText {
			if displayTest == choice {
				return bindSecret(ctx, c, app, secretName, secretChoiceName[i])
			}
		}
	}

	return createSecret(ctx, c, app, secretName)
}
