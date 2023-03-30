package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/acorn-io/acorn/pkg/mocks"
	"github.com/golang/mock/gomock"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVolume(t *testing.T) {
	tenYearsAgo := time.Now().AddDate(-10, 0, 0)

	defaultMockPreparation := func(f *mocks.MockClient) {
		f.EXPECT().VolumeList(gomock.Any()).Return(
			[]apiv1.Volume{{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: "volume",
					Labels: map[string]string{
						labels.AcornVolumeName: "vol",
						labels.AcornAppName:    "found",
					}},
				Spec:   apiv1.VolumeSpec{},
				Status: apiv1.VolumeStatus{AppName: "found", VolumeName: "vol"},
			}}, nil).AnyTimes()
		f.EXPECT().VolumeGet(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, name string) (*apiv1.Volume, error) {
				potentialVol := apiv1.Volume{TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{Name: "volume",
						Labels: map[string]string{
							labels.AcornVolumeName: "vol",
							labels.AcornAppName:    "found",
						}},
					Spec:   apiv1.VolumeSpec{},
					Status: apiv1.VolumeStatus{AppName: "found", VolumeName: "vol"},
				}

				switch name {
				case "dne":
					return nil, fmt.Errorf("error: volume %s does not exist", name)
				case "volume":
					return &potentialVol, nil
				case "found.vol":
					return &potentialVol, nil
				}
				return nil, nil
			}).AnyTimes()
		f.EXPECT().VolumeDelete(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, name string) (*apiv1.Volume, error) {
				switch name {
				case "dne":
					return nil, nil
				case "volume":
					return &apiv1.Volume{}, nil
				case "found.vol":
					return &apiv1.Volume{}, nil
				}
				return nil, nil
			}).AnyTimes()
	}

	type fields struct {
		Quiet  bool
		Output string
		All    bool
	}
	type args struct {
		cmd    *cobra.Command
		args   []string
		client *testdata.MockClient
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantOut string
		prepare func(f *mocks.MockClient)
	}{
		{
			name: "acorn volume", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			args: args{
				args:   []string{},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "ALIAS       NAME      APP-NAME   BOUND-VOLUME   CAPACITY   VOLUME-CLASS   STATUS    ACCESS-MODES   CREATED\nfound.vol   volume    found      vol            <nil>                                              292y ago\n",
		},
		{
			name: "acorn volume -o json", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			args: args{
				args:   []string{"-ojson"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "{\n    \"metadata\": {\n        \"name\": \"volume\",\n        \"creationTimestamp\": null\n    },\n    \"spec\": {},\n    \"status\": {\n        \"appName\": \"found\",\n        \"volumeName\": \"vol\",\n        \"columns\": {}\n    }\n}\n\n",
		},
		{
			name: "acorn volume -o yaml", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			args: args{
				args:   []string{"-oyaml"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "---\nmetadata:\n  creationTimestamp: null\n  name: volume\nspec: {}\nstatus:\n  appName: found\n  columns: {}\n  volumeName: vol\n\n",
		},
		{
			name: "acorn volume found.vol", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			args: args{
				args:   []string{"--", "found.vol"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "ALIAS       NAME      APP-NAME   BOUND-VOLUME   CAPACITY   VOLUME-CLASS   STATUS    ACCESS-MODES   CREATED\nfound.vol   volume    found      vol            <nil>                                              292y ago\n",
		},
		{
			name: "acorn volume dne", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			args: args{
				args:   []string{"--", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "error: volume dne does not exist",
		},
		{
			name: "acorn volume rm found.vol", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			args: args{
				args:   []string{"rm", "found.vol"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "found.vol\n",
		},
		{
			name: "acorn volume rm dne", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			args: args{
				args:   []string{"rm", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "Error: No such volume: dne\n",
		},
		{
			name: "acorn volume new inputs", fields: fields{},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().VolumeList(gomock.Any()).Return(
					[]apiv1.Volume{
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(tenYearsAgo),
								Name:              "my-volume",
								Labels: map[string]string{
									labels.AcornVolumeClass: "my-class",
									labels.AcornVolumeName:  "my-vol",
									labels.AcornAppName:     "app",
								},
							},
						},
					}, nil).AnyTimes()
			},
			args: args{
				args:   []string{},
				client: &testdata.MockClient{},
			},
			wantOut: "ALIAS        NAME        APP-NAME   BOUND-VOLUME   CAPACITY   VOLUME-CLASS   STATUS    ACCESS-MODES   CREATED\napp.my-vol   my-volume                             <nil>      my-class                                10y ago\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctrl := gomock.NewController(t)
			//Mocked client for cli's client calls.
			mClient := mocks.NewMockClient(ctrl)

			if tt.prepare != nil {
				tt.prepare(mClient)
			} else {
				defaultMockPreparation(mClient)
			}

			tt.args.cmd = NewVolume(CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader(""),
			})
			tt.args.cmd.SetArgs(tt.args.args)
			err := tt.args.cmd.Execute()
			if err != nil && !tt.wantErr {
				assert.Failf(t, "got err when err not expected", "got err: %s", err.Error())
			} else if err != nil && tt.wantErr {
				assert.Equal(t, tt.wantOut, err.Error())
			} else {
				w.Close()
				out, _ := io.ReadAll(r)
				assert.Equal(t, tt.wantOut, string(out))
			}
		})
	}
}
