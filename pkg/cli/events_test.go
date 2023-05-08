package cli

import (
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TODO(njhale): Add tests

func TestEvents(t *testing.T) {
	type fields struct {
		All bool
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
		// TODO(njhale): Add test cases
		//
		{
			name: "acorn events", fields: fields{
				All: false,
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
			// wantOut: "REPOSITORY   TAG       IMAGE-ID\ntesttag      latest    found-image1\ntesttag1     latest    found-image-\ntesttag2     v1        found-image-\n",
		}, //
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
			} else if err == nil && tt.wantErr {
				log.Fatal("got no err when err was expected")
			} else {
				w.Close()
				out, _ := io.ReadAll(r)
				testOut, _ := os.ReadFile(tt.wantOut)
				assert.Equal(t, string(testOut), string(out))
			}
		})
	}
}
