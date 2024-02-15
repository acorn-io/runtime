package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/appdefinition"
	"github.com/acorn-io/runtime/pkg/encryption/nacl"
	"github.com/acorn-io/z"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrJobNotDone  = errors.New("job not complete")
	ErrJobNoOutput = errors.New("job has no output")
)

const (
	Helper = "acorn-job-output-helper"
)

func GetJobOutputSecretName(namespace, jobName string) string {
	return name.SafeHashConcatName(jobName, "output", namespace)
}

// GetOutputFor obj must be acorn internal v1.Secret, v1.Service, or string
func GetOutputFor(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, name, serviceName string, obj interface{}) (err error) {
	data, err := getOutput(ctx, c, appInstance, name)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return fmt.Errorf("job [%s] produced no output in /run/secrets/output", name)
	}

	switch v := obj.(type) {
	case *string:
		*v = string(data)
	case *v1.Secret:
		if err := json.Unmarshal(data, v); err == nil && (v.Type != "" || len(v.Data) > 0) {
			return nil
		}
		appSpec, err := asAppSpec(data)
		if err != nil {
			return fmt.Errorf("failed to parse generated output for secret [%s] bytes [%d]: %v", serviceName, len(data), err)
		}
		secret, ok := appSpec.Secrets[serviceName]
		if !ok {
			return fmt.Errorf("generated output is missing secret [%s] bytes [%d]: %w", serviceName, len(data), appdefinition.ErrInvalidInput)
		}
		secret.DeepCopyInto(v)
	case *v1.Service:
		appSpec, err := asAppSpec(data)
		if err != nil {
			return fmt.Errorf("failed to parse generated output for service [%s] bytes [%d]: %v", serviceName, len(data), err)
		}
		svc, ok := appSpec.Services[serviceName]
		if !ok {
			return fmt.Errorf("generated output is missing service [%s] bytes [%d]: %w", serviceName, len(data), appdefinition.ErrInvalidInput)
		}
		svc.DeepCopyInto(v)
	default:
		return fmt.Errorf("invalid job output type %T", v)
	}

	return nil
}

func asAppSpec(data []byte) (*v1.AppSpec, error) {
	appDef, err := appdefinition.NewAppDefinition(data)
	if err != nil {
		return nil, err
	}
	return appDef.AppSpec()
}

func getOutput(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, name string) (data []byte, err error) {
	if _, ok := appInstance.Status.AppSpec.Jobs[name]; !ok {
		return nil, fmt.Errorf("generated output depends on undefined job [%s]", name)
	}

	defer func() {
		if err == nil {
			if nacl.IsAcornEncryptedData(data) {
				data, err = nacl.DecryptNamespacedData(ctx, c, data, appInstance.Namespace)
			}
		}
	}()

	secretName := GetJobOutputSecretName(appInstance.Status.Namespace, name)
	secret := &corev1.Secret{}

	if err := c.Get(ctx, router.Key(appInstance.Status.Namespace, secretName), secret); apierror.IsNotFound(err) {
		return nil, ErrJobNotDone
	} else if err != nil {
		return nil, err
	}

	if len(secret.Data["err"]) > 0 {
		return nil, errors.New(string(secret.Data["err"]))
	}

	if len(secret.Data["out"]) == 0 {
		return nil, ErrJobNoOutput
	}

	return secret.Data["out"], nil
}

// ShouldRunForEvent returns true if the job is configured to run for the given event.
func ShouldRunForEvent(eventName string, container v1.Container) bool {
	if len(container.Events) == 0 && container.Schedule == "" {
		// The default for non cronjobs is "create" and "update".
		return slices.Contains([]string{"create", "update"}, eventName)
	}
	return slices.Contains(container.Events, eventName)
}

// ShouldRun determines if the job should run based on the app's status.
func ShouldRun(jobName string, appInstance *v1.AppInstance) bool {
	for name, job := range appInstance.Status.AppSpec.Jobs {
		if name == jobName {
			return ShouldRunForEvent(GetEvent(jobName, appInstance), job)
		}
	}
	return false
}

// GetEvent determines the event type for the job based on the app's status.
func GetEvent(jobName string, appInstance *v1.AppInstance) string {
	if !appInstance.DeletionTimestamp.IsZero() {
		return "delete"
	}
	if z.Dereference(appInstance.Spec.Stop) {
		return "stop"
	}
	if (appInstance.Generation <= 1 || slices.Contains(appInstance.Status.AppSpec.Jobs[jobName].Events, "create")) && !appInstance.Status.AppStatus.Jobs[jobName].CreateEventSucceeded {
		// Create event jobs run at least once. So, if it hasn't succeeded, run it.
		return "create"
	}
	return "update"
}
