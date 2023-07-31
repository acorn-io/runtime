package apps

import (
	"context"
	"testing"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNestedValidateAppName(t *testing.T) {
	validator := &nestedValidator{}

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
		{
			name:        "Valid Name: Nested",
			appName:     "myapp.nested",
			expectValid: true,
		},
		{
			name:        "Valid Name: Multi Nested",
			appName:     "myapp.nested.nested",
			expectValid: true,
		},
		{
			name:        "Invalid Name: Nested Underscore",
			appName:     "myapp.my_app",
			expectValid: false,
		},
		{
			name:        "Invalid Name: Uppercase",
			appName:     "myapp.MyApp",
			expectValid: false,
		},
		{
			name:        "Invalid Name: Starts with number",
			appName:     "myapp.1app",
			expectValid: false,
		},
		{
			name:        "Invalid Name: Starts with dash",
			appName:     "myapp.-app",
			expectValid: false,
		},
		{
			name:        "Invalid Name: Ends with dash",
			appName:     "myapp.app-",
			expectValid: false,
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
