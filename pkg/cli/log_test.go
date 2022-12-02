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

func TestLog(t *testing.T) {
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
			name: "acorn logs found", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			wantOut: "",
		},
		{
			name: "acorn logs dne", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
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
			wantOut: "error: tag dne does not exist",
		},
		{
			name: "acorn logs found -n", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			},
			args: args{
				args:   []string{"-n-1", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "Tail: Invalid value: -1: must be greater than or equal to 0",
		},
		{
			name: "acorn logs found -n", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: client.CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			},
			args: args{
				args:   []string{"-n10", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "",
		},
	}
	for _, tt := range tests {
		r, w, _ := os.Pipe()
		os.Stdout = w
		tt.args.cmd = NewLogs(tt.commandContext)
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
