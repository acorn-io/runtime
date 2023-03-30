package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/acorn-io/acorn/pkg/prompt"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestAppRm(t *testing.T) {
	type fields struct {
		All   bool
		Type  []string
		Force bool
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
			name: "does not exist default type", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "deleting app dne: error: app dne does not exist",
		},
		{
			name: "does not exist force default type", fields: fields{
				All:   false,
				Type:  nil,
				Force: true,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			},
			args: args{
				args:   []string{"-f", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "deleting app dne: error: app dne does not exist",
		},
		{
			name: "does not exist container type", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-tc", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "",
		},
		{
			name: "does not exist volume type", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-tv", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "",
		},
		{
			name: "does exist default type", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found\n",
		},
		{
			name: "does exist force default type", fields: fields{
				All:   false,
				Type:  nil,
				Force: true,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			},
			args: args{
				args:   []string{"-f", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found\n",
		},
		{
			name: "does exist container type short", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-tc", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found.container\n",
		},
		{
			name: "does exist container type long", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-tcontainer", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found.container\n",
		},
		{
			name: "does exist app type short", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-ta", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found\n",
		},
		{
			name: "does exist app type long", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-tapp", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found\n",
		},
		{
			name: "does exist volume type short", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-tv", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: volume\n",
		},
		{
			name: "does exist volume type long", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-tvolume", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: volume\n",
		},
		{
			name: "does exist secret type short", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-ts", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found.secret\n",
		},
		{
			name: "does exist secret type long", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-tsecret", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found.secret\n",
		},
		{
			name: "does exist all type", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-a", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "Removed: found\nRemoved: volume\nRemoved: found.secret\n",
		},
		{
			name: "no app name arg default type", fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
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
			wantErr: true,
			wantOut: "requires at least 1 arg(s), only received 0",
		},
		{
			name: "no app name arg force default type", fields: fields{
				All:   false,
				Type:  nil,
				Force: true,
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			},
			args: args{
				args:   []string{"-f"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "requires at least 1 arg(s), only received 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewRm(tt.commandContext)
			tt.args.cmd.SetArgs(tt.args.args)
			prompt.NoPromptRemove = true
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
