package cli

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/cli/testdata"
	"github.com/acorn-io/runtime/pkg/mocks"
	"github.com/acorn-io/runtime/pkg/tables"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApp(t *testing.T) {
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
	var _, w, _ = os.Pipe()
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		wantOut        string
		commandContext CommandContext
	}{
		{
			name: "acorn app", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			},
			args: args{
				args:   []string{},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "NAME      IMAGE     HEALTHY   UP-TO-DATE   CREATED    ENDPOINTS   MESSAGE\nfound                                      292y ago               \n",
		},
		{
			name: "acorn app found", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			},
			args: args{
				args:   []string{"found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "NAME      IMAGE     HEALTHY   UP-TO-DATE   CREATED    ENDPOINTS   MESSAGE\nfound                                      292y ago               \n",
		},
		{
			name: "acorn app dne", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			},
			args: args{
				args:   []string{"dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "error: app dne does not exist",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewApp(tt.commandContext)
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

func TestWriteApp(t *testing.T) {
	ctrl := gomock.NewController(t)
	c := mocks.NewMockClient(ctrl)
	c.EXPECT().ImageList(gomock.Any()).Return(buildImageList(), nil).AnyTimes()

	cases := []struct {
		name           string
		appImageName   string
		appImageDigest string
		expected       string
	}{
		{
			name:           "basic",
			appImageName:   "myimage:latest",
			appImageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expected:       "NAME      IMAGE            HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS   MESSAGE\nmyapp     myimage:latest                          60m ago               \n",
		},
		{
			name:           "docker.io => index.docker.io",
			appImageName:   "docker.io/myimage:latest",
			appImageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			expected:       "NAME      IMAGE                      HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS   MESSAGE\nmyapp     docker.io/myimage:latest                          60m ago               \n",
		},
		{
			name:           "implicit docker.io",
			appImageName:   "myotherimage:latest",
			appImageDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			expected:       "NAME      IMAGE                 HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS   MESSAGE\nmyapp     myotherimage:latest                          60m ago               \n",
		},
		{
			name:           "tag moved",
			appImageName:   "myimage:latest",
			appImageDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			expected:       "NAME      IMAGE          HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS   MESSAGE\nmyapp     dddddddddddd                          60m ago               \n",
		},
		{
			name:           "not found",
			appImageName:   "dne:v1",
			appImageDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			expected:       "NAME      IMAGE          HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS   MESSAGE\nmyapp     111111111111                          60m ago               \n",
		},
		{
			name:           "no implicit assumption for other registries",
			appImageName:   "acornimage:latest",
			appImageDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			expected:       "NAME      IMAGE          HEALTHY   UP-TO-DATE   CREATED   ENDPOINTS   MESSAGE\nmyapp     eeeeeeeeeeee                          60m ago               \n",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			out := table.NewWriter(tables.App, false, "")

			app := &v1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myapp",
					CreationTimestamp: metav1.Time{
						Time: time.Now().Add(-1 * time.Hour),
					},
				},
				Status: internalv1.AppInstanceStatus{
					AppImage: internalv1.AppImage{
						Name:   tt.appImageName,
						Digest: tt.appImageDigest,
					},
				},
			}
			writeApp(context.Background(), app, out, c)
			out.Flush()
			w.Close()
			output, _ := io.ReadAll(r)
			assert.Equal(t, tt.expected, string(output))
		})
	}
}

func buildImageList() []v1.Image {
	return []v1.Image{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Tags:   []string{"myimage:latest"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
			Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Tags:   []string{"index.docker.io/myimage:latest", "myimage:v1"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			},
			Digest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Tags:   []string{"docker.io/myotherimage:latest"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			},
			Digest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			Tags:   nil,
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
			Digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			Tags:   []string{"acorn.io/acornimage:latest"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			},
			Digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			Tags:   []string{"acornimage:latest"},
		},
	}
}
