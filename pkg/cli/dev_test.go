package cli

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/runtime/pkg/cli/testdata"
	"github.com/acorn-io/runtime/pkg/mocks"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestDev(t *testing.T) {
	ctrl := gomock.NewController(t)
	mClient := mocks.NewMockClient(ctrl)
	mClient.EXPECT().AppList(gomock.Any()).Return(nil, nil)
	mClient.EXPECT().AppGet(gomock.Any(), "dne").
		Return(nil, fmt.Errorf("error: app dne does not exist")).AnyTimes()
	mClient.EXPECT().Info(gomock.Any()).
		Return(nil, nil).AnyTimes()
	mClient.EXPECT().ImageDetails(gomock.Any(), "image-dne", gomock.Any()).
		Return(nil, fmt.Errorf("✗  ERROR:  GET https://index.docker.io/v2/library/image-dne/manifests/latest: UNAUTHORIZED: authentication required; [map[Action:pull Class: Name:library/image-dne Type:repository]]")).AnyTimes()

	type fields struct {
		All   bool
		Type  []string
		Force bool
	}
	type args struct {
		cmd  *cobra.Command
		args []string
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
			args: args{
				args: []string{"image-dne"},
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			wantErr: true,
			wantOut: "✗  ERROR:  GET https://index.docker.io/v2/library/image-dne/manifests/latest: UNAUTHORIZED: authentication required; [map[Action:pull Class: Name:library/image-dne Type:repository]]",
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
