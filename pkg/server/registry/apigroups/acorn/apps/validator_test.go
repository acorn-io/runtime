package apps

import (
	"context"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
)

func TestCannotChangeAppRegion(t *testing.T) {
	validator := &Validator{}

	tests := []struct {
		name           string
		newApp, oldApp apiv1.App
	}{
		{
			name: "Cannot change region when initially set",
			oldApp: apiv1.App{
				Spec: internalv1.AppInstanceSpec{
					Region: "new-acorn-test-region",
				},
			},
			newApp: apiv1.App{
				Spec: internalv1.AppInstanceSpec{
					Region: "old-acorn-test-region",
				},
			},
		},
		{
			name: "Cannot change region from calculated default",
			oldApp: apiv1.App{
				Status: internalv1.AppInstanceStatus{
					Defaults: internalv1.Defaults{
						Region: "old-acorn-test-region",
					},
				},
			},
			newApp: apiv1.App{
				Spec: internalv1.AppInstanceSpec{
					Region: "new-acorn-test-region",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateUpdate(context.Background(), &tt.newApp, &tt.oldApp)
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
		})
	}
}
