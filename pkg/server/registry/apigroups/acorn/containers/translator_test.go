package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFromPublicName(t *testing.T) {
	app := &apiv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "appNs",
		},
		Status: v1.AppInstanceStatus{
			Namespace: "podNs",
		},
	}

	tests := []struct {
		containerPublicName      string
		containerPublicNamespace string

		expectedPodName      string
		expectedPodNamespace string
		expectedErr          error
	}{
		{
			containerPublicName:      "app.pod",
			containerPublicNamespace: app.Namespace,

			expectedPodName:      "pod",
			expectedPodNamespace: app.Status.Namespace,
			expectedErr:          nil,
		},
		{
			containerPublicName:      "app.pod:container",
			containerPublicNamespace: app.Namespace,

			expectedPodName:      "pod",
			expectedPodNamespace: app.Status.Namespace,
			expectedErr:          nil,
		},

		{
			containerPublicName:      "nonExistingApp.pod:container",
			containerPublicNamespace: app.Namespace,

			expectedPodName:      "nonExistingApp.pod:container",
			expectedPodNamespace: app.Namespace,
			expectedErr:          fmt.Errorf("\"nonExistingApp\" not found"),
		},
		{
			containerPublicName:      "app.pod:container",
			containerPublicNamespace: "nonExistingNamespace",

			expectedPodName:      "app.pod:container",
			expectedPodNamespace: "nonExistingNamespace",
			expectedErr:          fmt.Errorf("\"app\" not found"),
		},

		{
			containerPublicName:      "app", // incorrect format
			containerPublicNamespace: app.Namespace,

			expectedPodName:      "app",
			expectedPodNamespace: app.Namespace,
			expectedErr:          nil,
		},
		{
			containerPublicName:      "", // incorrect format
			containerPublicNamespace: app.Namespace,

			expectedPodName:      "",
			expectedPodNamespace: app.Namespace,
			expectedErr:          nil,
		},
	}

	for i := range tests {
		tc := tests[i]
		tcName := tc.containerPublicNamespace + "/" + tc.containerPublicName
		t.Run(tcName, func(t *testing.T) {
			//t.Parallel()

			req := tester.NewRequest(t, scheme.Scheme, app)
			translator := &Translator{req.Client}
			podNs, podName, err := translator.FromPublicName(context.Background(), tc.containerPublicNamespace, tc.containerPublicName)

			assert.Equal(t, tc.expectedPodName, podName)
			assert.Equal(t, tc.expectedPodNamespace, podNs)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}
