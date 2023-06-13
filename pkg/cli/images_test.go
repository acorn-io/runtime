package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImage(t *testing.T) {
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
			name: "acorn image", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID       REMOTE\ntesttag      latest    found-image1   \ntesttag1     latest    found-image-   \ntesttag2     v1        found-image-   \n",
		},
		{
			name: "acorn image --no-trunc", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"--no-trunc"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID                      REMOTE\ntesttag      latest    found-image1234567            \ntesttag1     latest    found-image-two-tags1234567   \ntesttag2     v1        found-image-two-tags1234567   \n",
		},
		{
			name: "acorn image -a", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-a"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID       REMOTE\ntesttag      latest    found-image1   \n<none>       <none>    found-image-   \ntesttag1     latest    found-image-   \ntesttag2     v1        found-image-   \n",
		},
		{
			name: "acorn image -c ", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-c"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID                      CONTAINER                      DIGEST\ntesttag      latest    found-image1234567            test-image-running-container   test-image-running-container\ntesttag1     latest    found-image-two-tags1234567   test-image-running-container   test-image-running-container\ntesttag2     v1        found-image-two-tags1234567   test-image-running-container   test-image-running-container\n",
		},
		{
			name: "acorn image -q ", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-q"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "found-image1234567\nfound-image-two-tags1234567\nfound-image-two-tags1234567\n",
		},
		{
			name: "acorn image -q -c", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-q", "-c"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "testtag:latest@test-image-running-container\ntesttag1:latest@test-image-running-container\ntesttag2:v1@test-image-running-container\n"},
		{
			name: "acorn image -q -a", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-q", "-a"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "found-image1234567\nfound-image-no-tag\nfound-image-two-tags1234567\nfound-image-two-tags1234567\n",
		},
		{
			name: "acorn image testtag", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageItem: &apiv1.Image{
						ObjectMeta: metav1.ObjectMeta{Name: "found-image1234567"},
						Tags:       []string{"testtag:latest"},
						Digest:     "1234567890asdfghkl"}},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"testtag"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID       REMOTE\ntesttag      latest    found-image1   \n",
		},
		{
			name: "acorn image testtag1", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageItem: &apiv1.Image{
						ObjectMeta: metav1.ObjectMeta{Name: "found-image-two-tags1234567"},
						Tags:       []string{"testtag1:latest", "testtag2:v1"},
						Digest:     "lkjhgfdsa1234567890",
					}},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"testtag1"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID       REMOTE\ntesttag1     latest    found-image-   \n",
		},
		{
			name: "acorn image digest", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageItem: &apiv1.Image{
						ObjectMeta: metav1.ObjectMeta{Name: "found-image-two-tags1234567"},
						Tags:       []string{"testtag1:latest", "testtag2:v1"},
						Digest:     "sha256:abcdef1234567890"}},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"abcdef1234567890"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID       REMOTE\ntesttag1     latest    found-image-   \ntesttag2     v1        found-image-   \n",
		},
		{
			name: "acorn image registry specific tag", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageItem: &apiv1.Image{
						ObjectMeta: metav1.ObjectMeta{Name: "registy1234567-two-tags"},
						Tags:       []string{"index.docker.io/subdir/test:v1", "index.docker.io/subdir/test:v2"},
						Digest:     "sha256:abcdef1234567890"}},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"index.docker.io/subdir/test:v1"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY                    TAG       IMAGE-ID       REMOTE\nindex.docker.io/subdir/test   v1        registy12345   \n",
		},
		{
			name: "acorn image digest multi tag", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageItem: &apiv1.Image{
						ObjectMeta: metav1.ObjectMeta{Name: "registy1234567-two-tags"},
						Tags:       []string{"index.docker.io/subdir/test:v1", "index.docker.io/subdir/test:v2"},
						Digest:     "sha256:abcdef1234567890"}},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"sha256:abcdef1234567890"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY                    TAG       IMAGE-ID       REMOTE\nindex.docker.io/subdir/test   v1        registy12345   \nindex.docker.io/subdir/test   v2        registy12345   \n",
		},
		{
			name: "acorn image -c digest multi tag", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageItem: &apiv1.Image{
						ObjectMeta: metav1.ObjectMeta{Name: "registy1234567-two-tags"},
						Tags:       []string{"index.docker.io/subdir/test:v1", "index.docker.io/subdir/test:v2"},
						Digest:     "sha256:abcdef1234567890"}},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-c", "sha256:abcdef1234567890"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY                    TAG       IMAGE-ID                  CONTAINER                      DIGEST\nindex.docker.io/subdir/test   v1        registy1234567-two-tags   test-image-running-container   test-image-running-container\n",
		},
		{
			name: "acorn image -q digest multi tag", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageItem: &apiv1.Image{
						ObjectMeta: metav1.ObjectMeta{Name: "registy1234567-two-tags"},
						Tags:       []string{"index.docker.io/subdir/test:v1", "index.docker.io/subdir/test:v2"},
						Digest:     "sha256:abcdef1234567890"}},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-q", "sha256:abcdef1234567890"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "registy1234567-two-tags\nregisty1234567-two-tags\n",
		},
		{
			name: "acorn image -q -c digest multi tag", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageItem: &apiv1.Image{
						ObjectMeta: metav1.ObjectMeta{Name: "registy1234567-two-tags"},
						Tags:       []string{"index.docker.io/subdir/test:v1", "index.docker.io/subdir/test:v2"},
						Digest:     "sha256:abcdef1234567890"}},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-q", "-c", "sha256:abcdef1234567890"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "index.docker.io/subdir/test:v1@test-image-running-container\n",
		},
		{
			name: "acorn image rm dne-image", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"rm", "dne-image"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "Error: No such image: dne-image\n",
		},
		{
			name: "acorn image rm ff12345", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"rm", "ff12345"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "deleting ff12345: unable to delete ff12345 (must be forced) - image is referenced in multiple repositories",
		},
		{
			name: "acorn image rm ff12345 -f", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"rm", "ff12345", "-f"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Untagged testtag1:latest\nUntagged testtag2:latest\nUntagged foo:v1\nUntagged foo:v2\nDeleted ff12345\n",
		},
		{
			name: "acorn image rm foo:v1", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"rm", "foo:v1"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Untagged foo:v1\n",
		},
		{
			name: "acorn image testtag remote", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					ImageList: []apiv1.Image{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "found-image1234567"},
							Tags:       []string{"testtag:latest"},
							Digest:     "1234567890asdfghkl",
							Remote:     true,
						},
					},
				},

				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"testtag"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID       REMOTE\ntesttag      latest    found-image1   *\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewImage(tt.commandContext)
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
