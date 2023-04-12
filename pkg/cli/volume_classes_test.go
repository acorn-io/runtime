package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVolumeClass(t *testing.T) {
	tests := []struct {
		name                  string
		existingVolumeClasses []apiv1.VolumeClass
		quiet                 bool
		args                  []string
		wantErr               bool
		wantOut               string
	}{
		{
			name: "acorn volume classes with one storage that is default",
			existingVolumeClasses: []apiv1.VolumeClass{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "local-path-class"},
					StorageClassName: "local-path",
					Default:          true,
					Size: adminv1.VolumeClassSize{
						Default: "5Gi",
						Min:     "1Gi",
						Max:     "10Gi",
					},
					Description:        "Just a little test",
					AllowedAccessModes: []v1.AccessMode{v1.AccessModeReadOnlyMany, v1.AccessModeReadWriteOnce},
				},
			},
			args:    []string{},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME               DEFAULT   INACTIVE   STORAGE-CLASS   SIZE-RANGE   DEFAULT-SIZE   ACCESS-MODES                   REGIONS   DESCRIPTION\nlocal-path-class   *                    local-path      1Gi-10Gi     5Gi            [readOnlyMany readWriteOnce]             Just a little test\n",
		},
		{
			name: "acorn volume classes with one storage that is not default",
			existingVolumeClasses: []apiv1.VolumeClass{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "local-path-class"},
					StorageClassName: "local-path",
					Size: adminv1.VolumeClassSize{
						Default: "3Gi",
						Min:     "0.5Gi",
						Max:     "5Gi",
					},
					AllowedAccessModes: []v1.AccessMode{v1.AccessModeReadWriteMany},
				},
			},
			args:    []string{},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME               DEFAULT   INACTIVE   STORAGE-CLASS   SIZE-RANGE   DEFAULT-SIZE   ACCESS-MODES      REGIONS   DESCRIPTION\nlocal-path-class                        local-path      0.5Gi-5Gi    3Gi            [readWriteMany]             \n",
		},
		{
			name: "acorn volume classes with two storages, one default",
			existingVolumeClasses: []apiv1.VolumeClass{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "local-path-class"},
					StorageClassName: "local-path",
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "magic-class"},
					StorageClassName: "magic",
					Default:          true,
				},
			},
			args:    []string{},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME               DEFAULT   INACTIVE   STORAGE-CLASS   SIZE-RANGE     DEFAULT-SIZE   ACCESS-MODES   REGIONS   DESCRIPTION\nlocal-path-class                        local-path      Unrestricted                                           \nmagic-class        *                    magic           Unrestricted                                           \n",
		},
		{
			name: "acorn volume classes with two storages, one inactive",
			existingVolumeClasses: []apiv1.VolumeClass{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "local-path-class"},
					StorageClassName: "local-path",
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "magic-class"},
					StorageClassName: "magic",
					Inactive:         true,
				},
			},
			args:    []string{},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME               DEFAULT   INACTIVE   STORAGE-CLASS   SIZE-RANGE     DEFAULT-SIZE   ACCESS-MODES   REGIONS   DESCRIPTION\nlocal-path-class                        local-path      Unrestricted                                           \nmagic-class                  *          magic           Unrestricted                                           \n",
		},
		{
			name: "acorn volume classes with two storages that are both default",
			existingVolumeClasses: []apiv1.VolumeClass{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "local-path-class"},
					StorageClassName: "local-path",
					Default:          true,
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "magic-class"},
					StorageClassName: "magic",
					Default:          true,
				},
			},
			args:    []string{},
			quiet:   false,
			wantErr: false,
			// Shouldn't happen, but we should handle it "correctly" if it does.
			wantOut: "NAME               DEFAULT   INACTIVE   STORAGE-CLASS   SIZE-RANGE     DEFAULT-SIZE   ACCESS-MODES   REGIONS   DESCRIPTION\nlocal-path-class   *                    local-path      Unrestricted                                           \nmagic-class        *                    magic           Unrestricted                                           \n",
		},
		{
			name: "acorn volume classes with arg that is default",
			existingVolumeClasses: []apiv1.VolumeClass{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "local-path-class"},
					StorageClassName: "local-path",
					Default:          true,
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "magic-class"},
					StorageClassName: "magic",
					Default:          true,
				},
			},
			args:    []string{"local-path-class"},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME               DEFAULT   INACTIVE   STORAGE-CLASS   SIZE-RANGE     DEFAULT-SIZE   ACCESS-MODES   REGIONS   DESCRIPTION\nlocal-path-class   *                    local-path      Unrestricted                                           \n",
		},
		{
			name: "acorn volume classes with arg that is not default",
			existingVolumeClasses: []apiv1.VolumeClass{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "local-path-class"},
					StorageClassName: "local-path",
					Default:          true,
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "magic-class"},
					StorageClassName: "magic",
				},
			},
			args:    []string{"magic-class"},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME          DEFAULT   INACTIVE   STORAGE-CLASS   SIZE-RANGE     DEFAULT-SIZE   ACCESS-MODES   REGIONS   DESCRIPTION\nmagic-class                        magic           Unrestricted                                           \n",
		},
		{
			name: "acorn volume classes with two storages with supported regions",
			existingVolumeClasses: []apiv1.VolumeClass{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "local-path-class"},
					StorageClassName: "local-path",
					SupportedRegions: []string{"local", "other-region"},
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "magic-class"},
					StorageClassName: "magic",
					Default:          true,
					SupportedRegions: []string{"local", "another-region"},
				},
			},
			args:    []string{},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME               DEFAULT   INACTIVE   STORAGE-CLASS   SIZE-RANGE     DEFAULT-SIZE   ACCESS-MODES   REGIONS                DESCRIPTION\nlocal-path-class                        local-path      Unrestricted                                 local,other-region     \nmagic-class        *                    magic           Unrestricted                                 local,another-region   \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			cmd := NewVolumeClasses(CommandContext{
				ClientFactory: &testdata.MockClientFactory{VolumeClassList: tt.existingVolumeClasses},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			})
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err != nil && !tt.wantErr {
				assert.Failf(t, "got err when err not expected", "got err: %s", err.Error())
			} else if err != nil && tt.wantErr {
				assert.Equal(t, tt.wantOut, err.Error())
			} else {
				assert.Nil(t, w.Close(), "error closing writer")
				out, _ := io.ReadAll(r)
				assert.Equal(t, tt.wantOut, string(out))
			}
		})
	}
}
