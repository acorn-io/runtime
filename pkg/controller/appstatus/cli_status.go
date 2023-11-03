package appstatus

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/publicname"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func CLIStatus(req router.Request, resp router.Response) (err error) {
	app := req.Object.(*v1.AppInstance)
	app.Status.Columns.UpToDate = uptodate(app)
	app.Status.Columns.Healthy = healthy(app)
	app.Status.Columns.Message = message(app)
	app.Status.Columns.Endpoints, err = endpoints(req, app)

	// There's clearly a better way to do this, but it works and I'm lazy. The intention is that we want
	// to detect that the acorn doesn't have any running containers (or needs to run containers) and has
	// produced whatever it needs to and the status is not really helpful anymore, because it's done.
	app.Status.AppStatus.Completed = strings.Contains(publicname.Get(app), ".") &&
		app.Status.Ready &&
		app.Status.Columns.Healthy == "0" &&
		app.Status.Columns.UpToDate == "0" &&
		app.Status.Columns.Message == "OK" &&
		app.Status.Columns.Endpoints == ""
	if app.Status.AppStatus.Completed {
		if parentName := app.Labels[labels.AcornParentAcornName]; parentName != "" {
			var parent v1.AppInstance
			if err := req.Get(&parent, app.Namespace, parentName); apierrors.IsNotFound(err) {
				app.Status.AppStatus.Completed = false
			} else if err != nil {
				return err
			}
		}
	}

	resp.Objects(app)
	return
}

func message(app *v1.AppInstance) string {
	buf := &bytes.Buffer{}
	if !app.DeletionTimestamp.IsZero() {
		buf.WriteString("removing")
	} else if app.GetStopped() && !app.Status.AppStatus.Stopped {
		buf.WriteString("stopping")
	} else if app.Status.ConfirmUpgradeAppImage != "" {
		buf.WriteString("Upgrade available: " + app.Status.ConfirmUpgradeAppImage)
	}

	for _, cond := range app.Status.Conditions {
		if cond.Type == v1.AppInstanceConditionReady {
			continue
		}
		if !cond.Success && (cond.Error || cond.Transitioning) && cond.Message != "" {
			if buf.Len() > 0 {
				buf.WriteString("; ")
			}
			if cond.Type == v1.AppInstanceConditionController && strings.HasPrefix(cond.Message, "[routes.go") {
				i := strings.Index(cond.Message, "] ")
				if i == -1 {
					buf.WriteString(cond.Message)
				} else {
					buf.WriteString(cond.Message[i+2:])
				}
			} else {
				buf.WriteString(cond.Message)
			}
		}
	}

	if buf.Len() != 0 {
		return buf.String()
	}

	if app.Status.Ready {
		return "OK"
	}

	return "pending"
}

func uptodate(app *v1.AppInstance) string {
	if app.Status.Namespace == "" {
		return "-"
	}
	if app.Status.AppStatus.Stopped {
		return "-"
	}
	var (
		desired, uptodate int32
	)
	for _, status := range app.Status.AppStatus.Containers {
		uptodate += status.UpToDateReplicaCount
		desired += status.RunningReplicaCount
	}
	if uptodate != desired {
		return fmt.Sprintf("%d/%d", uptodate, desired)
	}
	return strconv.Itoa(int(uptodate))
}

func healthy(app *v1.AppInstance) string {
	if app.Status.AppStatus.Stopped {
		return "stopped"
	}
	if app.Status.Namespace == "" {
		return "creating"
	}
	var (
		ready, desired int32
	)
	for _, status := range app.Status.AppStatus.Containers {
		desired += status.DesiredReplicaCount
		ready += status.ReadyReplicaCount
	}
	if ready != desired {
		return fmt.Sprintf("%d/%d", ready, desired)
	}
	return strconv.Itoa(int(ready))
}

func endpoints(req router.Request, app *v1.AppInstance) (string, error) {
	endpointTarget := map[string][]v1.Endpoint{}
	for _, endpoint := range app.Status.AppStatus.Endpoints {
		target := fmt.Sprintf("%s:%d", endpoint.Target, endpoint.TargetPort)
		endpointTarget[target] = append(endpointTarget[target], endpoint)
	}

	ingressTLSHosts, err := ingressTLSHosts(req.Ctx, req.Client, app)
	if err != nil {
		return "", err
	}

	var endpointStrings []string

	for _, endpoints := range typed.SortedValues(endpointTarget) {
		var publicStrings []string

		for _, endpoint := range endpoints {
			buf := &strings.Builder{}
			switch endpoint.Protocol {
			case v1.ProtocolHTTP:
				if !strings.HasPrefix(endpoint.Address, "http") {
					var host string
					a, b, ok := strings.Cut(endpoint.Address, "://")
					if ok {
						host, _, _ = strings.Cut(b, ":")
					} else {
						host, _, _ = strings.Cut(a, ":")
					}
					if _, ok := ingressTLSHosts[host]; ok {
						buf.WriteString("https://")
					} else {
						buf.WriteString("http://")
					}
				}

				if endpoint.Pending {
					buf.WriteString("<Pending Ingress>")
				} else {
					buf.WriteString(endpoint.Address)

					// Append the path if provided
					if len(endpoint.Path) > 0 {
						buf.WriteString(endpoint.Path)
					}
				}
			default:
				buf.WriteString(strings.ToLower(string(endpoint.Protocol)))
				buf.WriteString("://")

				if endpoint.Pending {
					buf.WriteString("<Pending Load Balancer>")
				} else {
					buf.WriteString(endpoint.Address)
				}
			}

			publicStrings = append(publicStrings, buf.String())
		}

		endpointStrings = append(endpointStrings, publicStrings...)
	}

	return strings.Join(endpointStrings, ", "), nil
}
