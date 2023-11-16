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
)

func Secrets(ctx context.Context, c client.Client, app *apiv1.App) error {
	for secretName, secret := range app.Status.AppSpec.Secrets {
		if strings.HasPrefix(secret.Type, v1.SecretTypeCredentialPrefix) {
			if err := loginSecret(ctx, c, app, secretName); err != nil {
				return err
			}
		}
	}

	//for _, acorn := range app.Status.AppStatus.Acorns {
	//	if acorn.AcornName != "" {
	//		if err := loginApp(ctx, c, app, acorn.AcornName); err != nil {
	//			return err
	//		}
	//	}
	//}
	//
	//for _, service := range app.Status.AppStatus.Services {
	//	if service.ServiceAcornName != "" {
	//		if err := loginApp(ctx, c, app, service.ServiceAcornName); err != nil {
	//			return err
	//		}
	//	}
	//}

	return nil
}

func secretIsOk(app *apiv1.App, secretName string) (string, bool) {
	return app.Status.AppStatus.Secrets[secretName].LinkOverride,
		!app.Status.AppStatus.Secrets[secretName].LoginRequired &&
			app.Status.AppStatus.Secrets[secretName].LinkOverride != ""
}

func printInstructions(app *apiv1.App, secretName string) error {
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

	fmt.Print(msg)
	fmt.Println()
	return nil
}

func bindSecret(ctx context.Context, c client.Client, app *apiv1.App, targetSecretName, overrideSecretName string) error {
	_, err := c.AppUpdate(ctx, app.Name, &client.AppUpdateOptions{
		Secrets: []v1.SecretBinding{
			{
				Secret: overrideSecretName,
				Target: targetSecretName,
			},
		},
	})
	return err
}

func createSecret(ctx context.Context, c client.Client, app *apiv1.App, secretName string) error {
	secretType := app.Status.AppSpec.Secrets[secretName].Type

	if err := printInstructions(app, secretName); err != nil {
		return err
	}

	data := map[string][]byte{}
	for _, key := range typed.SortedKeys(app.Status.AppSpec.Secrets[secretName].Data) {
		value, err := prompt.Password(key)
		if err != nil {
			return err
		}
		data[key] = value
	}

	secret, err := c.SecretCreate(ctx, secretName+"-", secretType, data)
	if err != nil {
		return err
	}

	return bindSecret(ctx, c, app, secretName, secret.Name)
}

func loginSecret(ctx context.Context, c client.Client, app *apiv1.App, secretName string) error {
	secretType := app.Status.AppSpec.Secrets[secretName].Type
	secretDisplayName := app.Name + "." + secretName

	if existing, ok := secretIsOk(app, secretName); ok {
		change, err := prompt.Bool(fmt.Sprintf("Credential [%s] is configured to [%s], do you want to change it",
			secretDisplayName, existing), false)
		if err != nil || !change {
			return err
		}
		fmt.Println()
	}

	secrets, err := c.SecretList(ctx)
	if err != nil {
		return err
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
		choice, err := prompt.Choice("Choose an existing credential or enter a new one", append(displayText, def), def)
		if err != nil {
			return err
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
