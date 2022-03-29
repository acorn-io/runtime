package cli

import (
	"bytes"
	"fmt"

	"github.com/ibuildthecloud/baaah/pkg/typed"
	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler-cli/pkg/table"
	"github.com/spf13/cobra"
)

func NewApp() *cobra.Command {
	return cli.Command(&App{}, cobra.Command{
		Use:     "app [flags] [APP_NAME...]",
		Aliases: []string{"apps", "a"},
		Example: `
herd app`,
		SilenceUsage: true,
		Short:        "List or get apps",
	})
}

type App struct {
	Quiet  bool   `desc:"Output only names" short:"q"`
	Output string `desc:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

func (a *App) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	out := table.NewWriter([][]string{
		{"NAME", "Name"},
		{"IMAGE", "Image"},
		{"HEALTHY", "{{ready .}}"},
		{"UPTODATE", "{{uptodate .}}"},
		{"CREATED", "{{ago .Created}}"},
		{"MESSAGE", "{{msg .}}"},
	}, "", a.Quiet, a.Output)

	out.AddFormatFunc("ready", ready)
	out.AddFormatFunc("uptodate", uptodate)
	out.AddFormatFunc("msg", message)

	apps, err := client.AppList(cmd.Context())
	if err != nil {
		return err
	}

	for _, app := range apps {
		out.Write(app)
	}

	return out.Err()
}

func message(app client.App) (string, error) {
	buf := &bytes.Buffer{}
	for _, entry := range typed.Sorted(app.Status.Conditions) {
		name, conn := entry.Key, entry.Value
		if !conn.Success && (conn.Error || conn.Transitioning) {
			if buf.Len() > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(name)
			buf.WriteString(": ")
			if conn.Message == "" {
				switch {
				case conn.Error:
					buf.WriteString("error")
				case conn.Transitioning:
					buf.WriteString("unknown")
				}
			} else {
				buf.WriteString(conn.Message)
			}
		}
	}
	return buf.String(), nil
}

func uptodate(app client.App) (interface{}, error) {
	if app.Status.Namespace == "" {
		return "creating", nil
	}
	if app.Status.Stopped {
		return "stopped", nil
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
		return fmt.Sprintf("%d/%d", uptodate, desired), nil
	}
	return uptodate, nil
}

func ready(app client.App) (interface{}, error) {
	if app.Status.Stopped || app.Status.Namespace == "" {
		return "-", nil
	}
	var (
		ready, desired int32
	)
	for _, status := range app.Status.ContainerStatus {
		desired += status.ReadyDesired
		ready += status.Ready
	}
	if ready != desired {
		return fmt.Sprintf("%d/%d", ready, desired), nil
	}
	return ready, nil
}
