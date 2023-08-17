package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acorn-io/runtime/pkg/cli/testdata"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredential(t *testing.T) {
	cfgDir, err := os.MkdirTemp("", "acorn-test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(cfgDir))
	}()
	acornConfig := filepath.Join(cfgDir, "acorn.yaml")

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
			name: "acorn credential found", fields: fields{
			All:    false,
			Quiet:  false,
			Output: "",
		},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					AcornConfig: acornConfig,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader(""),
			},
			args: args{
				args:   []string{"--", "test-server-address"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "SERVER                USERNAME   LOCAL\ntest-server-address              \n",
		},
		{
			name: "acorn credential dne", fields: fields{
			All:    false,
			Quiet:  false,
			Output: "",
		},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					AcornConfig: acornConfig,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader(""),
			},
			args: args{
				args:   []string{"--", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "error: cred dne does not exist",
		},
		{
			name: "acorn credential", fields: fields{
			All:    false,
			Quiet:  false,
			Output: "",
		},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					AcornConfig: acornConfig,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader(""),
			},
			args: args{
				args:   []string{},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "SERVER                USERNAME   LOCAL\ntest-server-address              \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewCredential(tt.commandContext)

			tt.args.cmd.SetArgs(tt.args.args)
			err := tt.args.cmd.Execute()
			if err != nil && !tt.wantErr {
				assert.Failf(t, "got err when err not expected", "got err: %s", err.Error())
			} else if err != nil && tt.wantErr {
				assert.Equal(t, tt.wantOut, err.Error())
			} else {
				require.NoError(t, w.Close())
				out, _ := io.ReadAll(r)
				assert.Equal(t, tt.wantOut, string(out))
			}
		})
	}
}
