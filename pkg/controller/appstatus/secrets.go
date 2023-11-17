package appstatus

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/ref"
	"github.com/acorn-io/runtime/pkg/secrets"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func linkedSecret(app *v1.AppInstance, name string) string {
	if name == "" {
		return ""
	}

	for _, binding := range app.Spec.Secrets {
		if binding.Target == name {
			return binding.Secret
		}
	}

	return ""
}

func (a *appStatusRenderer) readSecrets() error {
	existingStatus := a.app.Status.AppStatus.Secrets
	// reset state
	a.app.Status.AppStatus.Secrets = map[string]v1.SecretStatus{}

	for secretName, secretDef := range a.app.Status.AppSpec.Secrets {
		hash, err := configHash(secretDef)
		if err != nil {
			return err
		}

		s := v1.SecretStatus{
			Missing: existingStatus[secretName].Missing,
			CommonStatus: v1.CommonStatus{
				LinkOverride:          linkedSecret(a.app, secretName),
				ErrorMessages:         existingStatus[secretName].LookupErrors,
				TransitioningMessages: existingStatus[secretName].LookupTransitioning,
				ConfigHash:            hash,
			},
		}

		secret := &corev1.Secret{}
		if err := ref.Lookup(a.ctx, a.c, secret, a.app.Status.Namespace, secretName); apierrors.IsNotFound(err) {
			a.app.Status.AppStatus.Secrets[secretName] = s
			continue
		} else if err != nil {
			return err
		}

		s.UpToDate = secret.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation)) && secret.Annotations[labels.AcornConfigHashAnnotation] == hash
		s.Defined = true
		s.Ready = true

		sourceSecret := &corev1.Secret{}
		if err := a.c.Get(a.ctx, router.Key(secret.Labels[labels.AcornSecretSourceNamespace], secret.Labels[labels.AcornSecretSourceName]), sourceSecret); apierrors.IsNotFound(err) {
			s.State = "waiting"
			a.app.Status.AppStatus.Secrets[secretName] = s
			continue
		} else if err != nil {
			return err
		}

		s.SecretName = publicname.Get(sourceSecret)
		if secretDef.Type == string(v1.SecretTypeGenerated) && secretDef.Params.GetData()["job"] != "" {
			s.JobName = fmt.Sprint(secretDef.Params.GetData()["job"])
			s.JobReady, err = a.isJobReady(s.JobName)
			if err != nil {
				return err
			}
		} else {
			s.JobReady = true
		}

		s.Ready = s.Ready && s.JobReady
		s.DataKeys = typed.SortedKeys(sourceSecret.Data)

		a.app.Status.AppStatus.Secrets[secretName] = s
	}

	return nil
}

func setSecretMessages(ctx context.Context, c kclient.Client, app *v1.AppInstance) {
	for secretName, s := range app.Status.AppStatus.Secrets {
		// Not ready if we have any error messages
		if len(s.ErrorMessages) > 0 {
			s.Ready = false
		}

		if strings.HasPrefix(app.Status.AppSpec.Secrets[secretName].Type, v1.SecretTypeCredentialPrefix) {
			instructionsData := app.Status.AppSpec.Secrets[secretName].Params.GetData()["instructions"]
			if instructions, _ := instructionsData.(string); instructions != "" {
				result, err := secrets.NewInterpolator(ctx, c, app).Replace(instructions)
				if err == nil {
					s.LoginInstructions = result
				} else {
					s.LoginInstructions = instructions
				}
			}
		}

		if s.Ready {
			s.State = "created"
		} else if s.Missing && strings.HasPrefix(app.Status.AppSpec.Secrets[secretName].Type, v1.SecretTypeCredentialPrefix) {
			s.State = "pending"
			s.LoginRequired = true
			s.TransitioningMessages = []string{fmt.Sprintf("missing: \"acorn login %s\" required", publicname.Get(app))}
		} else if s.UpToDate {
			if len(s.ErrorMessages) > 0 {
				s.State = "failing"
			} else {
				s.State = "updating"
			}
		} else if s.Defined {
			if len(s.ErrorMessages) > 0 {
				s.State = "error"
			} else {
				s.State = "updating"
			}
		} else {
			if len(s.ErrorMessages) > 0 {
				s.State = "error"
			} else {
				s.State = "pending"
			}
		}

		app.Status.AppStatus.Secrets[secretName] = s
	}
}
