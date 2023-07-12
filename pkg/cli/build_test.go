package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/runtime/pkg/cli/testdata"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBuildBadTag(t *testing.T) {
	type args struct {
		cmd    *cobra.Command
		args   []string
		client *testdata.MockClient
	}
	var _, w, _ = os.Pipe()

	test := struct {
		name           string
		args           args
		wantErr        bool
		wantOut        string
		commandContext CommandContext
	}{
		name: "acorn build --tag -bad-tag",
		commandContext: CommandContext{
			ClientFactory: &testdata.MockClientFactory{},
			StdOut:        w,
			StdErr:        w,
			StdIn:         strings.NewReader("y\n"),
		},
		args: args{
			args:   []string{"--tag", "-bad-tag"},
			client: &testdata.MockClient{},
		},
		wantErr: true,
		wantOut: "invalid image tag: -bad-tag",
	}
	t.Run(test.name, func(t *testing.T) {
		test.args.cmd = NewBuild(test.commandContext)
		test.args.cmd.SetArgs(test.args.args)
		err := test.args.cmd.Execute()
		assert.Equal(t, test.wantOut, err.Error())
	})
}
