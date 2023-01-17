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

func TestSecret(t *testing.T) {
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
			name: "acorn secret", fields: fields{
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
			wantOut: "ALIAS         NAME           TYPE      KEYS      CREATED\nfound.found   found.secret             []        292y ago\n",
		},
		{
			name: "acorn secret -o json", fields: fields{
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
				args:   []string{"-ojson"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "{\n    \"metadata\": {\n        \"name\": \"found.secret\",\n        \"creationTimestamp\": null\n    }\n}\n\n",
		},
		{
			name: "acorn secret -o yaml", fields: fields{
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
				args:   []string{"-oyaml"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "---\nmetadata:\n  creationTimestamp: null\n  name: found.secret\n\n",
		},
		{
			name: "acorn secret found.secret", fields: fields{
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
				args:   []string{"--", "found.secret"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "ALIAS         NAME           TYPE      KEYS      CREATED\nfound.found   found.secret             []        292y ago\n",
		},
		{
			name: "acorn secret dne", fields: fields{
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
				args:   []string{"--", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "error: Secret dne does not exist",
		},
		{
			name: "acorn secret rm found.secret", fields: fields{
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
				args:   []string{"rm", "found.secret"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "found.secret\n",
		},
		{
			name: "acorn secret rm dne", fields: fields{
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
				args:   []string{"rm", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "Error: No such secret: dne\n",
		},
		{
			name: "acorn secret reveal found.secret", fields: fields{
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
				args:   []string{"reveal", "found.secret"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "NAME      TYPE      KEY       VALUE\n",
		},
		{
			name: "acorn secret reveal dne", fields: fields{
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
				args:   []string{"reveal", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "error: Secret dne does not exist",
		},
		{
			name: "acorn secret encrypt new.secret", fields: fields{
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
				args:   []string{"encrypt", "new.secret"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "ACORNENC:e30\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewSecret(tt.commandContext)
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
