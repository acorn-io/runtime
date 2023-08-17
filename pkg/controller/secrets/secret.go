package secrets

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/jobs"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/secrets"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type secEntry struct {
	name   string
	secret v1.Secret
}

func secretsOrdered(app *v1.AppInstance) (result []secEntry) {
	var generated []secEntry

	for _, entry := range typed.Sorted(app.Status.AppSpec.Secrets) {
		if entry.Value.Type == "generated" || entry.Value.Type == "template" {
			generated = append(generated, secEntry{name: entry.Key, secret: entry.Value})
		} else {
			result = append(result, secEntry{name: entry.Key, secret: entry.Value})
		}
	}
	return append(result, generated...)
}

func addSecretTransitioning(appInstance *v1.AppInstance, secretName, title, msg string) {
	c := appInstance.Status.AppStatus.Secrets[secretName]
	c.LookupTransitioning = append(c.LookupTransitioning, fmt.Sprintf("%s: [%s]", title, msg))
	appInstance.Status.AppStatus.Secrets[secretName] = c
}

func addSecretError(appInstance *v1.AppInstance, secretName string, err error) {
	c := appInstance.Status.AppStatus.Secrets[secretName]
	c.LookupErrors = append(c.LookupErrors, err.Error())
	appInstance.Status.AppStatus.Secrets[secretName] = c
}

func CreateSecrets(req router.Request, resp router.Response) (err error) {
	var (
		appInstance = req.Object.(*v1.AppInstance)
		allSecrets  = map[string]*corev1.Secret{}
	)

	if appInstance.Status.AppStatus.Secrets == nil {
		appInstance.Status.AppStatus.Secrets = map[string]v1.SecretStatus{}
	}

	for _, entry := range secretsOrdered(appInstance) {
		secretName := entry.name

		secret, err := secrets.GetOrCreateSecret(allSecrets, req, appInstance, secretName)
		if apierrors.IsNotFound(err) {
			if status := (*apierrors.StatusError)(nil); errors.As(err, &status) && status.ErrStatus.Details != nil {
				if status.ErrStatus.Details.Name != "" {
					addSecretTransitioning(appInstance, secretName, "missing", status.ErrStatus.Details.Name)
				}
			} else {
				addSecretTransitioning(appInstance, secretName, "missing", secretName)
			}
			continue
		} else if apiError := apierrors.APIStatus(nil); errors.As(err, &apiError) {
			addSecretError(appInstance, secretName, err)
			return nil
		} else if errors.Is(err, jobs.ErrJobNotDone) || errors.Is(err, jobs.ErrJobNoOutput) {
			addSecretTransitioning(appInstance, secretName, "waiting", err.Error())
			continue
		} else if err != nil {
			if strings.HasPrefix(err.Error(), "waiting") {
				addSecretTransitioning(appInstance, secretName, "waiting", err.Error())
			} else {
				addSecretError(appInstance, secretName, err)
			}
			continue
		}

		labelMap := map[string]string{
			labels.AcornAppName:               appInstance.Name,
			labels.AcornAppNamespace:          appInstance.Namespace,
			labels.AcornManaged:               "true",
			labels.AcornSecretName:            secretName,
			labels.AcornSecretSourceName:      secret.Name,
			labels.AcornSecretSourceNamespace: secret.Namespace,
		}
		labelMap = labels.Merge(labelMap, labels.GatherScoped(secretName, v1.LabelTypeSecret,
			appInstance.Status.AppSpec.Labels, entry.secret.Labels, appInstance.Spec.Labels))

		annotations := labels.GatherScoped(secretName, v1.LabelTypeSecret, appInstance.Status.AppSpec.Annotations,
			entry.secret.Annotations, appInstance.Spec.Annotations)

		annotations[labels.AcornAppGeneration] = strconv.FormatInt(appInstance.Generation, 10)

		resp.Objects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Namespace:   appInstance.Status.Namespace,
				Labels:      labelMap,
				Annotations: annotations,
			},
			Data: secret.Data,
			Type: secret.Type,
		})
	}

	return nil
}
