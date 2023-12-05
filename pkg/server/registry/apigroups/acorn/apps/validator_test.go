package apps

import (
	"context"
	"strings"
	"testing"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateAppName(t *testing.T) {
	validator := &Validator{}

	tests := []struct {
		name        string
		appName     string
		expectValid bool
	}{
		{
			name:        "Invalid Name: Underscore",
			appName:     "my_app",
			expectValid: false,
		},
		{
			name:        "Invalid Name: Uppercase",
			appName:     "MyApp",
			expectValid: false,
		},
		{
			name:        "Invalid Name: Starts with number",
			appName:     "1app",
			expectValid: false,
		},
		{
			name:        "Invalid Name: Starts with dash",
			appName:     "-app",
			expectValid: false,
		},
		{
			name:        "Invalid Name: Ends with dash",
			appName:     "app-",
			expectValid: false,
		},
		{
			name:        "Valid Name: Lowercase",
			appName:     "myapp",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &apiv1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.appName,
				},
			}
			err := validator.ValidateName(context.Background(), app)
			if tt.expectValid && err != nil {
				t.Fatalf("Expected valid, got error: %v", err)
			}
			if !tt.expectValid && err == nil {
				t.Fatalf("Expected error, got nil")
			}
		})
	}
}

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
					ResolvedOfferings: internalv1.ResolvedOfferings{
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

func TestCannotUpdateNestedAcorn(t *testing.T) {
	validator := &Validator{}

	oldApp := apiv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myapp.nested",
			Labels: map[string]string{
				labels.AcornParentAcornName: "myapp",
			},
		},
	}
	newApp := apiv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myapp.nested",
			Labels: map[string]string{
				labels.AcornParentAcornName: "myapp",
			},
		},
	}

	if err := validator.ValidateUpdate(context.Background(), &newApp, &oldApp); err == nil {
		t.Fatalf("Expected error, got no error")
	} else {
		assert.True(t, len(err) > 0)
		assert.True(t, strings.Contains(err[0].Error(), "update the parent Acorn"))
	}
}
