package volumes

import (
	"context"
	"fmt"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	corev1 "k8s.io/api/core/v1"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFromPublicName(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "appNs",
		},
		Status: v1.AppInstanceStatus{
			Namespace: "podNs",
		},
	}

	volume := &apiv1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvcName",
			Namespace: "ns",
			Labels: map[string]string{
				labels.AcornAppName:    "app",
				labels.AcornVolumeName: "volName",
			},
		},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvcName",
			Namespace: "ns",
			Labels: map[string]string{
				labels.AcornAppName:    "app",
				labels.AcornVolumeName: "volName",
			},
		},
	}

	tests := []struct {
		volumePublicName string
		volumePublicNS   string

		expectedVolumeName string
		expectedErr        error
	}{
		{
			volumePublicName: "app.volName",
			volumePublicNS:   "ns",

			expectedVolumeName: "pvcName",
			expectedErr:        nil,
		},
		{
			volumePublicName: "app.volName.unknownName",
			volumePublicNS:   "ns",

			expectedVolumeName: "app.volName.unknownName",
			expectedErr:        fmt.Errorf("failed to find pv name from alias: app.volName.unknownName"),
		},
	}

	for i := range tests {
		tc := tests[i]
		tcName := tc.volumePublicName + " to " + tc.expectedVolumeName
		t.Run(tcName, func(t *testing.T) {
			req := tester.NewRequest(t, scheme.Scheme, pv, volume, app)
			translator := &Translator{req.Client}
			volumeNs, volumeName, err := translator.FromPublicName(context.Background(), tc.volumePublicNS, tc.volumePublicName)

			assert.Equal(t, tc.expectedVolumeName, volumeName)
			assert.Equal(t, "", volumeNs) // Will never be a namespace for volume
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}
