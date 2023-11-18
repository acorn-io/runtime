package appstatus

import (
	"fmt"
	"strconv"
	"strings"

	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	client2 "github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/ref"
	"github.com/acorn-io/z"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *appStatusRenderer) readServices() error {
	existingStatus := a.app.Status.AppStatus.Services

	// reset state
	a.app.Status.AppStatus.Services = make(map[string]v1.ServiceStatus, len(a.app.Status.AppSpec.Services))

	for _, entry := range typed.Sorted(a.app.Status.AppSpec.Services) {
		serviceName, serviceDef := entry.Key, entry.Value
		hash, err := configHash(serviceDef)
		if err != nil {
			return err
		}

		s := v1.ServiceStatus{
			CommonStatus: v1.CommonStatus{
				LinkOverride: ports.LinkService(a.app, serviceName),
				ConfigHash:   hash,
			},
			ExpressionErrors:           existingStatus[serviceName].ExpressionErrors,
			MissingConsumerPermissions: existingStatus[serviceName].MissingConsumerPermissions,
		}

		s.Ready, s.Defined, err = a.isServiceReady(serviceName)
		if err != nil {
			return err
		}

		if s.LinkOverride == "" && serviceDef.Image != "" {
			s.ServiceAcornName = publicname.ForChild(a.app, serviceName)
			serviceAcorn := &v1.AppInstance{}
			err := a.c.Get(a.ctx, router.Key(a.app.Namespace, name2.SafeHashConcatName(a.app.Name, serviceName)), serviceAcorn)
			if apierrors.IsNotFound(err) || err == nil {
				s.ServiceAcornName = publicname.Get(serviceAcorn)
				s.ServiceAcornReady = serviceAcorn.Status.Ready && serviceAcorn.Annotations[labels.AcornConfigHashAnnotation] == hash
				if serviceAcorn.Status.AppStatus.LoginRequired {
					a.app.Status.AppStatus.LoginRequired = true
				}
			} else {
				return err
			}
		}

		if len(s.MissingConsumerPermissions) > 0 {
			s.ErrorMessages = append(s.ErrorMessages, "invalid service is trying to grant permissions to consumer not granted"+
				" to this service. The acorn containing this service must be updated to request the following permissions, "+z.Pointer(client2.ErrRulesNeeded{
				Permissions: s.MissingConsumerPermissions,
			}).Error())
		}

		service := &v1.ServiceInstance{}
		if err := ref.Lookup(a.ctx, a.c, service, a.app.Status.Namespace, serviceName); apierrors.IsNotFound(err) {
			s.Ready = false
			s.UpToDate = s.ServiceAcornName != ""
			s.Defined = s.ServiceAcornName != ""
			a.app.Status.AppStatus.Services[serviceName] = s
		} else if err != nil {
			return err
		} else {
			s.Defined = s.Defined || !service.Status.HasService
			s.UpToDate = service.Namespace != a.app.Status.Namespace ||
				service.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation))
			s.UpToDate = s.Defined && s.UpToDate && (s.ServiceAcornReady || s.LinkOverride != "" || serviceDef.External != "" || service.Annotations[labels.AcornConfigHashAnnotation] == hash)
			s.Ready = (s.Ready || !service.Status.HasService) && s.UpToDate
			if s.ServiceAcornName != "" {
				s.Ready = s.Ready && s.ServiceAcornReady
			}

			s.Default = service.Spec.Default
			s.Ports = service.Spec.Ports
			s.Data = service.Spec.Data
			s.Consumer = service.Spec.Consumer
			s.Secrets = service.Spec.Secrets
			s.Address = service.Spec.Address
			if len(service.Status.Endpoints) > 0 {
				s.Endpoint = service.Status.Endpoints[0].Address
			}

			var (
				failed         bool
				failedName     string
				failedMessage  string
				waiting        bool
				waitingName    string
				waitingMessage string
			)

			for _, condition := range service.Status.Conditions {
				if condition.Error {
					failed = true
					failedName = service.Name
					failedMessage = condition.Message
				} else if condition.Transitioning || !condition.Success {
					waiting = true
					waitingName = service.Name
					waitingMessage = condition.Message
				}
			}

			switch {
			case failed:
				s.ErrorMessages = append(s.ErrorMessages, fmt.Sprintf("%s: failed [%s]", failedName, failedMessage))
			case waiting:
				s.TransitioningMessages = append(s.TransitioningMessages, fmt.Sprintf("%s: waiting [%s]", waitingName, waitingMessage))
			default:
			}
		}

		// The ref.Lookup call above will find what the service resolves to, but not the actual local service object.
		// This is because the ref.Lookup will traverse spec.external references until it finds the destination.
		// Here we lookup the local object so we can determine if it's the default or not
		localService := &v1.ServiceInstance{}
		if err := a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, serviceName), localService); err == nil {
			s.Default = localService.Spec.Default
		} else if !apierrors.IsNotFound(err) {
			return err
		}

		a.app.Status.AppStatus.Services[serviceName] = s
	}
	return nil
}

func setServiceMessages(app *v1.AppInstance) {
	for serviceName, s := range app.Status.AppStatus.Services {
		addExpressionErrors(&s.CommonStatus, s.ExpressionErrors)

		// Not ready if we have any error messages
		if len(s.ErrorMessages) > 0 {
			s.Ready = false
		}

		if s.Ready {
			s.State = "ready"
		} else if s.UpToDate {
			if len(s.ErrorMessages) > 0 {
				s.State = "failing"
			} else if s.ServiceAcornName != "" && !s.ServiceAcornReady {
				s.State = "not ready"
				s.TransitioningMessages = append(s.TransitioningMessages, fmt.Sprintf("acorn [%s] is not ready", s.ServiceAcornName))
			} else {
				s.State = "not ready"
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

		app.Status.AppStatus.Services[serviceName] = s
	}
}

func (a *appStatusRenderer) isServiceReady(svc string) (ready bool, found bool, err error) {
	return a.isServiceReadyByNamespace(nil, a.app.Status.Namespace, svc)
}

func (a *appStatusRenderer) isServiceReadyByNamespace(seen []client.ObjectKey, namespace, svc string) (ready bool, found bool, err error) {
	if slices.Contains(seen, router.Key(namespace, svc)) {
		return false, false, fmt.Errorf("circular service dependency on %s/%s: %v", namespace, svc, seen)
	}
	seen = append(seen, router.Key(namespace, svc))

	var svcDep corev1.Service
	if err = a.c.Get(a.ctx, router.Key(namespace, svc), &svcDep); apierrors.IsNotFound(err) {
		return false, false, nil
	} else if err != nil {
		// if err just return it as not ready
		return false, true, err
	}

	if svcDep.Labels[labels.AcornManaged] != "true" {
		// for services we don't manage, just return ready always
		return true, true, nil
	}

	if svcDep.Spec.ExternalName != "" {
		cfg, err := config.Get(a.ctx, a.c)
		if err != nil {
			// if err just return it as not ready
			return false, true, nil
		}
		if strings.HasSuffix(svcDep.Spec.ExternalName, cfg.InternalClusterDomain) {
			parts := strings.Split(svcDep.Spec.ExternalName, ".")
			if len(parts) > 2 {
				return a.isServiceReadyByNamespace(seen, parts[1], parts[0])
			}
		}
		// for unknown external names we just assume they are always ready
		return true, true, nil
	}

	var endpoints corev1.Endpoints
	err = a.c.Get(a.ctx, router.Key(namespace, svc), &endpoints)
	if apierrors.IsNotFound(err) {
		return false, false, nil
	} else if err != nil {
		return false, true, err
	}

	for _, subset := range endpoints.Subsets {
		if len(subset.Addresses) > 0 {
			return true, true, nil
		}
	}

	return false, true, nil
}
