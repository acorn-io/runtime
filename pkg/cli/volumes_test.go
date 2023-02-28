package cli

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVolume(t *testing.T) {
	tenYearsAgo := time.Now().AddDate(-10, 0, 0)
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
			name: "acorn volume", fields: fields{
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
				args:   []string{},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "NAME           APP-NAME   BOUND-VOLUME   CAPACITY   VOLUME-CLASS   STATUS    ACCESS-MODES   CREATED\nfound.volume   found      found.volume   <nil>                                              292y ago\n",
		},
		{
			name: "acorn volume -o json", fields: fields{
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
				args:   []string{"-ojson"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "{\n    \"metadata\": {\n        \"name\": \"found.volume\",\n        \"creationTimestamp\": null\n    },\n    \"spec\": {},\n    \"status\": {\n        \"appName\": \"found\",\n        \"volumeName\": \"found.volume\",\n        \"columns\": {}\n    }\n}\n\n",
		},
		{
			name: "acorn volume -o yaml", fields: fields{
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
				args:   []string{"-oyaml"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "---\nmetadata:\n  creationTimestamp: null\n  name: found.volume\nspec: {}\nstatus:\n  appName: found\n  columns: {}\n  volumeName: found.volume\n\n",
		},
		{
			name: "acorn volume found.volume", fields: fields{
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
				args:   []string{"--", "found.volume"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "NAME           APP-NAME   BOUND-VOLUME   CAPACITY   VOLUME-CLASS   STATUS    ACCESS-MODES   CREATED\nfound.volume   found      found.volume   <nil>                                              292y ago\n",
		},
		{
			name: "acorn volume dne", fields: fields{
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
				args:   []string{"--", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "error: volume dne does not exist",
		},
		{
			name: "acorn volume rm found.volume", fields: fields{
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
				args:   []string{"rm", "found.volume"},
				client: &testdata.MockClient{},
			},
			wantErr: false,
			wantOut: "found.volume\n",
		},
		{
			name: "acorn volume rm dne", fields: fields{
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
				args:   []string{"rm", "dne"},
				client: &testdata.MockClient{},
			},
			wantErr: true,
			wantOut: "Error: No such volume: dne\n",
		},
		{
			name: "acorn volume", fields: fields{},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactory{
					VolumeList: []apiv1.Volume{
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(tenYearsAgo),
								Name:              "my-volume",
								Labels: map[string]string{
									labels.AcornVolumeClass: "my-class",
								},
							},
						},
					},
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader(""),
			},
			args: args{
				args:   []string{},
				client: &testdata.MockClient{},
			},
			wantOut: "NAME        APP-NAME   BOUND-VOLUME   CAPACITY   VOLUME-CLASS   STATUS    ACCESS-MODES   CREATED\nmy-volume                             <nil>      my-class                                10y ago\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewVolume(tt.commandContext)
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
