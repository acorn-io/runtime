package usage

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/version"
	"github.com/acorn-io/z"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type RuntimeComponent string
type RuntimeAction string

const (
	BaseURL = ""

	ComponentCLI        RuntimeComponent = "cli"
	ComponentAPIServer  RuntimeComponent = "api-server"
	ComponentController RuntimeComponent = "controller"

	ActionInstall RuntimeAction = "install" // subpaths: /install/{version}

	ActionHeartbeat RuntimeAction = "heartbeat" // subpaths: /heartbeat/{version}

	EnvUsageMetrics = "ACORN_USAGE_METRICS"
)

func Pulse(ctx context.Context, c kclient.Client, component RuntimeComponent, action RuntimeAction, elements ...string) error {
	if !UsageMetricsEnabled(ctx, c) {
		return nil
	}

	url := fmt.Sprintf("%s/%s/%s", BaseURL, component, action)
	for _, e := range elements {
		url += "/" + e
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func Heartbeat(ctx context.Context, c kclient.Client, component RuntimeComponent, interval time.Duration, elements ...string) {
	wait.UntilWithContext(ctx, func(_ context.Context) {
		err := Pulse(ctx, c, component, ActionHeartbeat, elements...)
		if err != nil {
			logrus.Warnf("failed to send heartbeat for %q: %v", component, err)
		}
	}, interval)
}

// UsageMetricsEnabled returns true if usage metrics are enabled.
// Usage Metrics are enabled by default, but are disabled by
// a) setting the ACORN_USAGE_METRICS environment variable to "disabled"
// b) setting the DisableUsageMetrics field in the acorn config to true
// c) running a development build (dirty or tag ending in -dev)
// d) running in an unofficial build (BaseURL is empty)
func UsageMetricsEnabled(ctx context.Context, c kclient.Client) bool {
	enabled := true
	if c != nil {
		cfg, err := config.Get(ctx, c)
		if err != nil {
			return false
		}
		enabled = !z.Dereference(cfg.DisableUsageMetrics)
	}
	return enabled && os.Getenv(EnvUsageMetrics) != "disabled" && !version.Get().Dirty && !strings.HasSuffix(version.Get().Tag, "-dev") && BaseURL != ""
}
