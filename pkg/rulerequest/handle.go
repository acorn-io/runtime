package rulerequest

import (
	"context"
	"errors"
	"fmt"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/prompt"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/pterm/pterm"
)

func PromptRun(ctx context.Context, c client.Client, dangerous bool, image string, opts client.AppRunOptions) (*apiv1.App, error) {
	app, err := c.AppRun(ctx, image, &opts)
	if permErr := (*client.ErrRulesNeeded)(nil); errors.As(err, &permErr) {
		if ok, promptErr := handleDangerous(dangerous, permErr.Permissions); promptErr != nil {
			return nil, fmt.Errorf("%s: %w", promptErr.Error(), err)
		} else if ok {
			opts.Permissions = permErr.Permissions
			app, err = c.AppRun(ctx, image, &opts)
		}
	}
	return app, err
}

func PromptUpdate(ctx context.Context, c client.Client, dangerous bool, name string, opts client.AppUpdateOptions) (*apiv1.App, error) {
	app, err := c.AppUpdate(ctx, name, &opts)
	if permErr := (*client.ErrRulesNeeded)(nil); errors.As(err, &permErr) {
		if ok, promptErr := handleDangerous(dangerous, permErr.Permissions); promptErr != nil {
			return nil, fmt.Errorf("%s: %w", promptErr.Error(), err)
		} else if ok {
			opts.Permissions = permErr.Permissions
			app, err = c.AppUpdate(ctx, name, &opts)
		}
	}
	return app, err
}

func handleDangerous(dangerous bool, perms []v1.Permissions) (bool, error) {
	if dangerous {
		return true, nil
	}

	requests := ToRuleRequests(perms)

	pterm.Warning.Println(
		`This application would like to request the following runtime permissions.
This could be VERY DANGEROUS to the cluster if you do not trust this
application. If you are unsure say no.`)
	pterm.Println()

	writer := table.NewWriter(tables.RuleRequests, "", false, "")
	for _, request := range requests {
		writer.Write(request)
	}

	if err := writer.Close(); err != nil {
		return false, err
	}

	pterm.Println()
	return prompt.Bool("Do you want to allow this app to have these (POTENTIALLY DANGEROUS) permissions?", false)
}
