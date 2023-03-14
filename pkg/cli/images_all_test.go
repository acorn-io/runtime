package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestImages(t *testing.T) {
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
			// So -A flag doesn't actually do anything in the CLI
			// However, to ensure this remains the case, we check that a response from multi-client
			// which would return images across multiple namespaces,
			name: "acorn images -a -A", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {

				f.EXPECT().ImageList(gomock.Any()).Return(
					[]apiv1.Image{
						{TypeMeta: metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "image-1-9102390",
								Namespace: "NameOne",
							},
							Tags:   []string{"testtag:latest"},
							Digest: "1234567890asdfghkl",
						},
						{TypeMeta: metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "image-2-2198031",
								Namespace: "NameTwo",
							},
						},
					}, nil)
			},
			args: args{
				args: []string{"-a"},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID\ntesttag      latest    image-1-9102\n<none>       <none>    image-2-2198\n",
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
			tt.args.cmd = NewImage(CommandContext{
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
