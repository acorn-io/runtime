package appstatus

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
)

func CLIStatus(req router.Request, resp router.Response) (err error) {
	app := req.Object.(*v1.AppInstance)
	app.Status.Columns.UpToDate = uptodate(app)
	app.Status.Columns.Healthy = healthy(app)
	app.Status.Columns.Message = message(app)
	app.Status.Columns.Endpoints, err = endpoints(req, app)
	resp.Objects(app)
	return
}

func message(app *v1.AppInstance) string {
	buf := &bytes.Buffer{}
	for _, cond := range app.Status.Conditions {
		if cond.Type == v1.AppInstanceConditionReady {
			continue
		}
		if !cond.Success && (cond.Error || cond.Transitioning) && cond.Message != "" {
			if buf.Len() > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString("[")
			buf.WriteString(cond.Type)
			buf.WriteString(": ")
			buf.WriteString(cond.Message)
			buf.WriteString("]")
		}
	}

	if buf.Len() != 0 {
		return buf.String()
	}
	if app.Status.ConfirmUpgradeAppImage != "" {
		return "Upgrade available: " + app.Status.ConfirmUpgradeAppImage
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
		desired += status.DesiredReplicaCount
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

	for _, entry := range typed.Sorted(endpointTarget) {
		var (
			target, endpoints = entry.Key, entry.Value
			publicStrings     []string
		)

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
			default:
				buf.WriteString(strings.ToLower(string(endpoint.Protocol)))
				buf.WriteString("://")
			}

			if endpoint.Pending {
				if endpoint.Protocol == "http" {
					buf.WriteString("<Pending Ingress>")
				} else {
					buf.WriteString("<Pending Load Balancer>")
				}
			} else {
				buf.WriteString(endpoint.Address)
			}
			publicStrings = append(publicStrings, buf.String())
		}

		endpointStrings = append(endpointStrings,
			fmt.Sprintf("%s => %s", strings.Join(publicStrings, " | "), target))
	}

	return strings.Join(endpointStrings, ", "), nil
}
