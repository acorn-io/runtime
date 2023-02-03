package cli

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/mocks"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestInfo(t *testing.T) {
	type fields struct {
		Quiet  bool
		Output string
		All    bool
	}
	type args struct {
		cmd  *cobra.Command
		args []string
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
			name: "acorn info", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().Info(gomock.Any()).Return(
					[]apiv1.Info{
						{TypeMeta: metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "NameOne",
								Namespace: "NameOne",
							},
							Spec: apiv1.InfoSpec{
								Tag: "OneTag",
							},
						},
					}, nil)
			},
			args: args{
				args: []string{},
			},
			wantErr: false,
			wantOut: "./testdata/info/info_test.txt",
		},
		{
			name: "acorn info empty response", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().Info(gomock.Any()).Return(
					nil, nil)
			},
			args: args{
				args: []string{},
			},
			wantErr: false,
			wantOut: "./testdata/info/info_test_empty.txt",
		},
		{
			name: "acorn info -A", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			// Want to return two entries
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().Info(gomock.Any()).Return(
					[]apiv1.Info{
						{TypeMeta: metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "NameOne",
								Namespace: "NameOne",
							},
							Spec: apiv1.InfoSpec{
								Tag: "OneTag",
							},
						},
						{TypeMeta: metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "NameTwo",
								Namespace: "NameTwo",
							},
							Spec: apiv1.InfoSpec{
								Tag: "TwoTag",
							},
						},
					}, nil)
			},
			args: args{
				args: []string{},
			},
			wantErr: false,
			wantOut: "./testdata/info/info_test-a.txt",
		},
		{
			name: "acorn info -o yaml", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().Info(gomock.Any()).Return([]apiv1.Info{
					{TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "NameOne",
							Namespace: "NameOne",
						},
						Spec: apiv1.InfoSpec{
							Tag: "OneTag",
						},
					},
				}, nil)
			},
			args: args{
				args: []string{"-oyaml"},
			},
			wantErr: false,
			wantOut: "./testdata/info/info_test.txt",
		},
		{
			name: "acorn info -o json", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().Info(gomock.Any()).Return(
					[]apiv1.Info{
						{TypeMeta: metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "NameOne",
								Namespace: "NameOne",
							},
							Spec: apiv1.InfoSpec{
								Tag: "OneTag",
							},
						},
					}, nil)
			},
			args: args{
				args: []string{"-ojson"},
			},
			wantErr: false,
			wantOut: "./testdata/info/info_test_json.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			//Mocked client for cli's client calls.
			mClient := mocks.NewMockClient(ctrl)
			if tt.prepare != nil {
				tt.prepare(mClient)
			}

			r, w, _ := os.Pipe()
			os.Stdout = w
			// Mock client factory just returns the gomock client.
			tt.args.cmd = NewInfo(CommandContext{
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
				testOut, _ := os.ReadFile(tt.wantOut)
				assert.Equal(t, string(testOut), string(out))
			}
		})
	}
}
