package appdefinition

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/ibuildthecloud/baaah/pkg/typed"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
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
	for _, entry := range typed.Sorted(app.Status.Conditions) {
		name, conn := entry.Key, entry.Value
		if !conn.Success && (conn.Error || conn.Transitioning) && conn.Message != "" {
			if buf.Len() > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString("[")
			buf.WriteString(name)
			buf.WriteString(": ")
			buf.WriteString(conn.Message)
			buf.WriteString("]")
		}
	}
	if buf.Len() == 0 {
		return "OK"
	}
	return buf.String()
}

func uptodate(app *v1.AppInstance) string {
	if app.Status.Namespace == "" {
		return "-"
	}
	if app.Status.Stopped {
		return "-"
	}
	var (
		ready, desired, uptodate int32
	)
	for _, status := range app.Status.ContainerStatus {
		uptodate += status.UpToDate
		desired += status.ReadyDesired
		ready += status.Ready
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
