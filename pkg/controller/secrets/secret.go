package secrets

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/jobs"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/secrets"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
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

func CreateSecrets(req router.Request, resp router.Response) (err error) {
	var (
		missing     []string
		errored     []string
		waiting     []string
		appInstance = req.Object.(*v1.AppInstance)
		allSecrets  = map[string]*corev1.Secret{}
		cond        = condition.Setter(appInstance, resp, v1.AppInstanceConditionSecrets)
	)

	defer func() {
		if err != nil {
			cond.Error(err)
			return
		}

		buf := strings.Builder{}
		if len(missing) > 0 {
			sort.Strings(missing)
			buf.WriteString("missing: [")
			buf.WriteString(strings.Join(missing, ", "))
			buf.WriteString("]")
		}
		if len(errored) > 0 {
			sort.Strings(errored)
			if buf.Len() > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString("errored: [")
			buf.WriteString(strings.Join(errored, ", "))
			buf.WriteString("]")
		}
		if len(waiting) > 0 {
			sort.Strings(waiting)
			if buf.Len() > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString("waiting: [")
			buf.WriteString(strings.Join(waiting, ", "))
			buf.WriteString("]")
		}

		if buf.Len() > 0 {
			cond.Error(errors.New(buf.String()))
		} else {
			cond.Success()
		}
	}()

	for _, entry := range secretsOrdered(appInstance) {
		secretName := entry.name
		secret, err := secrets.GetOrCreateSecret(allSecrets, req, appInstance, secretName)
		if apierrors.IsNotFound(err) {
			if status := (*apierrors.StatusError)(nil); errors.As(err, &status) && status.ErrStatus.Details != nil {
				if status.ErrStatus.Details.Name != "" {
					missing = append(missing, status.ErrStatus.Details.Name)
				}
			} else {
				missing = append(missing, secretName)
			}
			continue
		} else if apiError := apierrors.APIStatus(nil); errors.As(err, &apiError) {
			cond.Error(err)
			return err
		} else if errors.Is(err, jobs.ErrJobNotDone) || errors.Is(err, jobs.ErrJobNoOutput) {
			waiting = append(waiting, fmt.Sprintf("%s: %v", secretName, err))
			continue
		} else if err != nil {
			if strings.HasPrefix(err.Error(), "waiting") {
				waiting = append(waiting, fmt.Sprintf("%s: %v", secretName, err))
			} else {
				errored = append(errored, fmt.Sprintf("%s: %v", secretName, err))
			}
			continue
		}

		labelMap := map[string]string{
			labels.AcornAppName:      appInstance.Name,
			labels.AcornAppNamespace: appInstance.Namespace,
			labels.AcornManaged:      "true",
		}
		labelMap = labels.Merge(labelMap, labels.GatherScoped(secretName, v1.LabelTypeSecret,
			appInstance.Status.AppSpec.Labels, entry.secret.Labels, appInstance.Spec.Labels))

		annotations := labels.GatherScoped(secretName, v1.LabelTypeSecret, appInstance.Status.AppSpec.Annotations,
			entry.secret.Annotations, appInstance.Spec.Annotations)

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
