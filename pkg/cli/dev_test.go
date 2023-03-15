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

func TestDev(t *testing.T) {
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
			name: "acorn dev image-dne", fields: fields{
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
				args:   []string{"image-dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "error: app image-dne does not exist",
		},
		{
			name: "acorn dev --name dne . ", fields: fields{
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
				args:   []string{"--name", "dne", "."},
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
			tt.args.cmd = NewDev(tt.commandContext)
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
