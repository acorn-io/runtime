package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestInstall(t *testing.T) {
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
		wantOut        []string
		commandContext client.CommandContext
	}{
		{
			name: "acorn install: Valid",
			fields: fields{
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
				args:   []string{},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: []string{"✔  Installation done"},
		},
		{
			name: "acorn install --http-endpoint-pattern: Valid",
			fields: fields{
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
				args:   []string{"--http-endpoint-pattern", "{{.App}}-{{.Container}}.{{.ClusterDomain}}"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: []string{"✔  Installation done"},
		},
		{
			name: "acorn install --http-endpoint-pattern: Invalid",
			fields: fields{
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
				args:   []string{"--http-endpoint-pattern", "{{.Invalid}}-{{.HttpEndpoint}}.{{.Pattern}}"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: []string{"endpoint pattern is invalid"},
		},
		{
			name: "acorn install with --cluster-domains: Valid",
			fields: fields{
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
				args:   []string{"--cluster-domain", "foo.bar.com"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: []string{"✔  Installation done"},
		},
		{
			name: "acorn install --lets-encrypt enabled: Valid",
			fields: fields{
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
				args:   []string{"--lets-encrypt", "enabled", "--lets-encrypt-tos-agree", "--lets-encrypt-email=foo@bar.com"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: []string{
				"•  You've enabled automatic TLS certificate provisioning with Let's Encrypt",
				"✔  Installation done"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewInstall(tt.commandContext)
			tt.args.cmd.SetArgs(tt.args.args)
			err := tt.args.cmd.Execute()
			var out []byte
			w.Close()
			if err != nil {
				if !tt.wantErr {
					assert.Failf(t, "got err when err not expected", "got err: %s", err.Error())
				}
				out = []byte(err.Error())
			} else {
				out, _ = io.ReadAll(r)
			}

			for _, wantOut := range tt.wantOut {
				testOut, err := os.ReadFile(wantOut)
				if err == os.ErrNotExist {
					testOut = []byte(wantOut)
				}
				assert.Contains(t, string(out), string(testOut))
			}
		})
	}
}
