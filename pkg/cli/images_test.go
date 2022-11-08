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
			wantOut: "REPOSITORY   TAG       IMAGE-ID\ntestrepo     testtag   found-image1\n",
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
			wantOut: "REPOSITORY   TAG       IMAGE-ID\ntestrepo     testtag   found-image1234567\n",
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
			wantOut: "REPOSITORY   TAG       IMAGE-ID\ntestrepo     testtag   found-image1\n<none>       <none>    found-image-\n",
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
			wantOut: "REPOSITORY   TAG       IMAGE-ID             CONTAINER                      DIGEST\ntestrepo     testtag   found-image1234567   test-image-running-container   test-image-running-container\n",
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
			wantOut: "found-image1234567\n",
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
			wantOut: "testrepo:testtag@test-image-running-container\n",
		},
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
			wantOut: "found-image1234567\nfound-image-no-tag\n",
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
