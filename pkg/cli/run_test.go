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

func TestRunArgs_Env(t *testing.T) {
	os.Setenv("x222", "y333")
	runArgs := RunArgs{
		Env: []string{"x222", "y=1"},
	}
	opts, err := runArgs.ToOpts()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "x222", opts.Env[0].Name)
	assert.Equal(t, "y333", opts.Env[0].Value)
	assert.Equal(t, "y", opts.Env[1].Name)
	assert.Equal(t, "1", opts.Env[1].Value)
}

func TestRunMemory(t *testing.T) {
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
			name: "acorn run -h", fields: fields{
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
				args:   []string{"-h"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "./testdata/run/acorn_run_help.txt",
		},
		{
			name: "acorn run -m found.container=256Miii found ", fields: fields{
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
				args:   []string{"-m found.container=256Miii", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "invalid number \"256Miii\"",
		},
		{
			name: "acorn run -m found.container=notallowed found ", fields: fields{
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
				args:   []string{"-m found.container=notallowed", "found"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "illegal number start \"notallowed\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewRun(tt.commandContext)
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
