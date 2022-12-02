package cli

import (
	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
	"testing"
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
		commandContext client.CommandContext
	}{
		{
			name: "acorn image", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			wantOut: "REPOSITORY   TAG       IMAGE-ID\ntesttag      latest    found-image1\ntesttag1     latest    found-image-\ntesttag2     latest    found-image-\n",
		},
		{
			name: "acorn image --no-trunc", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			wantOut: "REPOSITORY   TAG       IMAGE-ID\ntesttag      latest    found-image1234567\ntesttag1     latest    found-image-two-tags1234567\ntesttag2     latest    found-image-two-tags1234567\n",
		},
		{
			name: "acorn image -a", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			wantOut: "REPOSITORY   TAG       IMAGE-ID\ntesttag      latest    found-image1\n<none>       <none>    found-image-\ntesttag1     latest    found-image-\ntesttag2     latest    found-image-\n",
		},
		{
			name: "acorn image -c ", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			wantOut: "REPOSITORY   TAG       IMAGE-ID                      CONTAINER                      DIGEST\ntesttag      latest    found-image1234567            test-image-running-container   test-image-running-container\ntesttag1     latest    found-image-two-tags1234567   test-image-running-container   test-image-running-container\ntesttag2     latest    found-image-two-tags1234567   test-image-running-container   test-image-running-container\n",
		},
		{
			name: "acorn image -q ", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			commandContext: client.CommandContext{
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
			wantOut: "testtag:latest@test-image-running-container\ntesttag1:latest@test-image-running-container\ntesttag2:latest@test-image-running-container\n"},
		{
			name: "acorn image -q -a", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			name: "acorn image rm found-image1234567", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"rm", "found-image1234567"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "found-image1234567\n",
		},
		{
			name: "acorn image rm dne-image", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			name: "acorn image rm found-image-two-tags1234567", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"rm", "found-image-two-tags1234567"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "deleting found-image-two-tags1234567: unable to delete found-image-two-tags1234567 (must be forced) - image is referenced in multiple repositories",
		},
		{
			name: "acorn image rm found-image-two-tags1234567 -f", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"rm", "found-image-two-tags1234567", "-f"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "found-image-two-tags1234567\n",
		},
	}
	for _, tt := range tests {
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
	}
}

func TestParseRepo(t *testing.T) {
	type args struct {
		tag string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "repo latest tag",
			args: args{
				tag: "testrepo/testimage:latest",
			},
			want: "testrepo/testimage",
		},
		{
			name: "no repo",
			args: args{
				tag: "testimage:latest",
			},
			want: "testimage",
		},
		{
			name: "repo named tag",
			args: args{
				tag: "testrepo/testimage:v1",
			},
			want: "testrepo/testimage",
		},
		{
			name: "no tag",
			args: args{
				tag: "",
			},
			want: "<none>",
		},
		{
			name: "repo with tag",
			args: args{
				tag: "testrepo:v1/testimage:v1",
			},
			want: "testrepo:v1/testimage",
		},
		{
			name: "multi branch repo image tag",
			args: args{
				tag: "testrepo/subdirectory/testimage:latest",
			},
			want: "testrepo/subdirectory/testimage",
		},
		{
			name: "multi branch repo tag and image tag",
			args: args{
				tag: "testrepo/subdirectory:v1/testimage:latest",
			},
			want: "testrepo/subdirectory:v1/testimage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, parseRepo(tt.args.tag), "parseRepo(%v)", tt.args.tag)
		})
	}
}
func TestParseTag(t *testing.T) {
	type args struct {
		tag string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "repo latest tag",
			args: args{
				tag: "testrepo/testimage:latest",
			},
			want: "latest",
		},
		{
			name: "no repo",
			args: args{
				tag: "testimage:latest",
			},
			want: "latest",
		},
		{
			name: "repo named tag",
			args: args{
				tag: "testrepo/testimage:v1",
			},
			want: "v1",
		},
		{
			name: "no tag",
			args: args{
				tag: "",
			},
			want: "<none>",
		},
		{
			name: "repo with tag",
			args: args{
				tag: "testrepo:v1/testimage:v1",
			},
			want: "v1",
		},
		{
			name: "multi branch repo no tag",
			args: args{
				tag: "testrepo/subdirectory/testimage",
			},
			want: "<none>",
		},
		{
			name: "multi branch repo image tag",
			args: args{
				tag: "testrepo/subdirectory/testimage:latest",
			},
			want: "latest",
		},
		{
			name: "multi branch repo tag and image tag",
			args: args{
				tag: "testrepo/subdirectory:v1/testimage:latest",
			},
			want: "latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, parseTag(tt.args.tag), "parseTag(%v)", tt.args.tag)
		})
	}
}
func TestIncludesTag(t *testing.T) {
	type args struct {
		tag string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "repo latest tag",
			args: args{
				tag: "testrepo/testimage:latest",
			},
			want: false,
		},
		{
			name: "no repo",
			args: args{
				tag: "testimage:latest",
			},
			want: true,
		},
		{
			name: "repo named tag",
			args: args{
				tag: "testrepo/testimage:v1",
			},
			want: false,
		},
		{
			name: "repo with tag",
			args: args{
				tag: "testrepo:v1/testimage:v1",
			},
			want: false,
		},
		{
			name: "no tag",
			args: args{
				tag: "",
			},
			want: false,
		},
		{
			name: "repo no tag",
			args: args{
				tag: "testrepo/testimage",
			},
			want: false,
		},
		{
			name: "multi branch repo no tag",
			args: args{
				tag: "testrepo/subdirectory/testimage",
			},
			want: false,
		},
		{
			name: "multi branch repo image tag",
			args: args{
				tag: "testrepo/subdirectory/testimage:latest",
			},
			want: false,
		},
		{
			name: "multi branch repo tag and image tag",
			args: args{
				tag: "testrepo/subdirectory:v1/testimage:latest",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, includesTag(tt.args.tag), "includesTag(%v)", tt.args.tag)
		})
	}
}
func TestIncludesRepoAndTag(t *testing.T) {
	type args struct {
		tag string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "repo latest tag",
			args: args{
				tag: "testrepo/testimage:latest",
			},
			want: true,
		},
		{
			name: "no repo",
			args: args{
				tag: "testimage:latest",
			},
			want: false,
		},
		{
			name: "repo named tag",
			args: args{
				tag: "testrepo/testimage:v1",
			},
			want: true,
		},
		{
			name: "repo with tag",
			args: args{
				tag: "testrepo:v1/testimage:v1",
			},
			want: true,
		},
		{
			name: "no tag",
			args: args{
				tag: "",
			},
			want: false,
		},
		{
			name: "repo no tag",
			args: args{
				tag: "testrepo/testimage",
			},
			want: false,
		},
		{
			name: "multi branch repo no tag",
			args: args{
				tag: "testrepo/subdirectory/testimage",
			},
			want: false,
		},
		{
			name: "multi branch repo image tag",
			args: args{
				tag: "testrepo/subdirectory/testimage:latest",
			},
			want: true,
		},
		{
			name: "multi branch repo tag and image tag",
			args: args{
				tag: "testrepo/subdirectory:v1/testimage:latest",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, includesRepoAndTag(tt.args.tag), "includesRepoAndTag(%v)", tt.args.tag)
		})
	}
}
