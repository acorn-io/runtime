package appdefinition

import (
	"bytes"
	"fmt"
	"strconv"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
)

func CLIStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	app.Status.Columns.UpToDate = uptodate(app)
	app.Status.Columns.Healthy = healthy(app)
	app.Status.Columns.Message = message(app)
	resp.Objects(app)
	return nil
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
	if app.Status.Stopped {
		return "-"
	}
	var (
		desired, uptodate int32
	)
	for _, status := range app.Status.ContainerStatus {
		uptodate += status.UpToDate
		desired += status.ReadyDesired
	}
	if uptodate != desired {
		return fmt.Sprintf("%d/%d", uptodate, desired)
	}
	return strconv.Itoa(int(uptodate))
}

func healthy(app *v1.AppInstance) string {
	if app.Status.Stopped {
		return "stopped"
	}
	if app.Status.Namespace == "" {
		return "creating"
	}
	var (
		ready, desired int32
	)
	for _, status := range app.Status.ContainerStatus {
		desired += status.ReadyDesired
		ready += status.Ready
	}
	if ready != desired {
		return fmt.Sprintf("%d/%d", ready, desired)
	}
	return strconv.Itoa(int(ready))
}
