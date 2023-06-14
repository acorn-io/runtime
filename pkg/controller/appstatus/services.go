package appstatus

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/ref"
	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (a *appStatusRenderer) readServices() error {
	existingStatus := a.app.Status.AppStatus.Services

	// reset state
	a.app.Status.AppStatus.Services = make(map[string]v1.ServiceStatus, len(a.app.Status.AppSpec.Services))

	for serviceName, serviceDef := range a.app.Status.AppSpec.Services {
		s := v1.ServiceStatus{
			CommonStatus: v1.CommonStatus{
				LinkOverride: ports.LinkService(a.app, serviceName),
			},
			ExpressionErrors: existingStatus[serviceName].ExpressionErrors,
		}

		s.Ready, s.Defined = a.isServiceReady(serviceName)

		if s.LinkOverride == "" && serviceDef.Image != "" {
			s.ServiceAcornName = publicname.ForChild(a.app, serviceName)
			serviceAcorn := &v1.AppInstance{}
			err := a.c.Get(a.ctx, router.Key(a.app.Namespace, name2.SafeHashConcatName(a.app.Name, serviceName)), serviceAcorn)
			if apierrors.IsNotFound(err) || err == nil {
				s.ServiceAcornName = publicname.Get(serviceAcorn)
				s.ServiceAcornReady = serviceAcorn.Status.Ready
			} else {
				return err
			}
		}

		service := &v1.ServiceInstance{}
		if err := ref.Lookup(a.ctx, a.c, service, a.app.Status.Namespace, serviceName); apierrors.IsNotFound(err) {
			s.Ready = false
			s.UpToDate = s.ServiceAcornName != ""
			s.Defined = s.ServiceAcornName != ""
			a.app.Status.AppStatus.Services[serviceName] = s
			continue
		} else if err != nil {
			return err
		}

		s.Defined = s.Defined || !service.Status.HasService
		s.UpToDate = service.Namespace != a.app.Status.Namespace ||
			service.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation))
		s.Ready = (s.Ready || !service.Status.HasService) && s.UpToDate
		if s.ServiceAcornName != "" {
			s.Ready = s.Ready && s.ServiceAcornReady
		}

		s.Default = service.Spec.Default
		s.Ports = service.Spec.Ports
		s.Data = service.Spec.Data
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

		addExpressionErrors(&s.CommonStatus, s.ExpressionErrors)

		a.app.Status.AppStatus.Services[serviceName] = s
	}
	return nil
}

func (a *appStatusRenderer) isServiceReady(svc string) (ready bool, found bool) {
	return a.isServiceReadyByNamespace(a.app.Status.Namespace, svc)
}

func (a *appStatusRenderer) isServiceReadyByNamespace(namespace, svc string) (ready bool, found bool) {
	var svcDep corev1.Service
	err := a.c.Get(a.ctx, router.Key(namespace, svc), &svcDep)
	if apierrors.IsNotFound(err) {
		return false, false
	}
	if err != nil {
		// if err just return it as not ready
		return false, true
	}

	if svcDep.Labels[labels.AcornManaged] != "true" {
		// for services we don't manage, just return ready always
		return true, true
	}

	if svcDep.Spec.ExternalName != "" {
		cfg, err := config.Get(a.ctx, a.c)
		if err != nil {
			// if err just return it as not ready
			return false, true
		}
		if strings.HasSuffix(svcDep.Spec.ExternalName, cfg.InternalClusterDomain) {
			parts := strings.Split(svcDep.Spec.ExternalName, ".")
			if len(parts) > 2 {
				return a.isServiceReadyByNamespace(parts[1], parts[0])
			}
		}
		// for unknown external names we just assume they are always ready
		return true, true
	}

	var endpoints corev1.Endpoints
	err = a.c.Get(a.ctx, router.Key(namespace, svc), &endpoints)
	if apierrors.IsNotFound(err) {
		return false, false
	}
	if err != nil {
		// if err just return it as not ready
		return false, true
	}

	for _, subset := range endpoints.Subsets {
		if len(subset.Addresses) > 0 {
			return true, true
		}
	}

	return false, true
}
