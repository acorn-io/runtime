package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestContainer(t *testing.T) {
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
			name: "acorn container", fields: fields{
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
			wantOut: "NAME              APP       IMAGE     STATE     RESTARTCOUNT   CREATED    MESSAGE\nfound.container                                 0              292y ago   \n",
		},
		{
			name: "acorn container found.container", fields: fields{
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
				args:   []string{"--", "found.container"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "NAME              APP       IMAGE     STATE     RESTARTCOUNT   CREATED    MESSAGE\nfound.container                                 0              292y ago   \n",
		},
		{
			name: "acorn container found", fields: fields{
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
				args:   []string{"--", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "NAME              APP       IMAGE     STATE     RESTARTCOUNT   CREATED    MESSAGE\nfound.container                                 0              292y ago   \n",
		},
		{
			name: "acorn container kill found", fields: fields{
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
				args:   []string{"kill", "found.container"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "found.container\n",
		},
		{
			name: "acorn container kill dne", fields: fields{
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
				args:   []string{"kill", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "Error: No such container: dne\n",
		},
	}
	for _, tt := range tests {
		r, w, _ := os.Pipe()
		os.Stdout = w
		tt.args.cmd = NewContainer(tt.commandContext)
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
