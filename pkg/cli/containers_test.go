package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/mocks"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// create test data
	mockContainer1 = &apiv1.ContainerReplica{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "found.container1",
			CreationTimestamp: metav1.NewTime(time.Now().AddDate(-10, 0, 0)),
		},
		Spec:   apiv1.ContainerReplicaSpec{AppName: "found"},
		Status: apiv1.ContainerReplicaStatus{},
	}
	mockContainer2 = &apiv1.ContainerReplica{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "found.container2",
			CreationTimestamp: metav1.NewTime(time.Now().AddDate(-10, 0, 0)),
		},
		Spec: apiv1.ContainerReplicaSpec{AppName: "found"},
		Status: apiv1.ContainerReplicaStatus{
			Columns: apiv1.ContainerReplicaColumns{
				State: "stopped",
			},
		},
	}
	mockApp = &apiv1.App{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "found",
			CreationTimestamp: metav1.NewTime(time.Now().AddDate(-10, 0, 0)),
		},
		Spec:   v1.AppInstanceSpec{Secrets: []v1.SecretBinding{{Secret: "found.secret", Target: "found"}}},
		Status: v1.AppInstanceStatus{Ready: true},
	}
)

func TestContainer(t *testing.T) {
	// create mock client and declare all expected function calls
	ctrl := gomock.NewController(t)
	mClient := mocks.NewMockClient(ctrl)
	mClient.EXPECT().ContainerReplicaGet(gomock.Any(), "found.container1").
		Return(mockContainer1, nil).AnyTimes()
	mClient.EXPECT().ContainerReplicaGet(gomock.Any(), "found.container2").
		Return(mockContainer2, nil).AnyTimes()
	mClient.EXPECT().ContainerReplicaGet(gomock.Any(), "dne").
		Return(nil, fmt.Errorf("error: container dne does not exist")).AnyTimes()
	mClient.EXPECT().ContainerReplicaList(gomock.Any(), nil).
		Return([]apiv1.ContainerReplica{*mockContainer1, *mockContainer2}, nil).AnyTimes()
	mClient.EXPECT().ContainerReplicaList(gomock.Any(), &client.ContainerReplicaListOptions{App: "found"}).
		Return([]apiv1.ContainerReplica{*mockContainer1, *mockContainer2}, nil).AnyTimes()
	mClient.EXPECT().ContainerReplicaDelete(gomock.Any(), "found.container1").
		Return(mockContainer1, nil).AnyTimes()
	mClient.EXPECT().ContainerReplicaDelete(gomock.Any(), "dne").
		Return(nil, fmt.Errorf("error: No such container: dne"))
	mClient.EXPECT().AppGet(gomock.Any(), "found").Return(mockApp, nil).AnyTimes()
	mClient.EXPECT().AppGet(gomock.Any(), gomock.Not("found")).Return(nil, fmt.Errorf("app not found"))

	type fields struct {
		Quiet  bool
		Output string
		All    bool
	}
	type args struct {
		cmd    *cobra.Command
		args   []string
		client *mocks.MockClient
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
			name: "acorn container", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{},
				client: mClient,
			},
			wantErr: false,
			wantOut: "NAME               APP       IMAGE     STATE     RESTARTCOUNT   CREATED   MESSAGE\nfound.container1                                 0              10y ago   \n",
		},
		{
			name: "acorn container -a", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-a"},
				client: mClient,
			},
			wantErr: false,
			wantOut: "NAME               APP       IMAGE     STATE     RESTARTCOUNT   CREATED   MESSAGE\nfound.container1                                 0              10y ago   \nfound.container2                       stopped   0              10y ago   \n",
		},
		{
			name: "acorn container found.container1", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"--", "found.container1"},
				client: mClient,
			},
			wantErr: false,
			wantOut: "NAME               APP       IMAGE     STATE     RESTARTCOUNT   CREATED   MESSAGE\nfound.container1                                 0              10y ago   \n",
		},
		{
			name: "acorn container found", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"--", "found"},
				client: mClient,
			},
			wantErr: false,
			wantOut: "NAME               APP       IMAGE     STATE     RESTARTCOUNT   CREATED   MESSAGE\nfound.container1                                 0              10y ago   \n",
		},
		{
			name: "acorn container found -a", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"-a", "--", "found"},
				client: mClient,
			},
			wantErr: false,
			wantOut: "NAME               APP       IMAGE     STATE     RESTARTCOUNT   CREATED   MESSAGE\nfound.container1                                 0              10y ago   \nfound.container2                       stopped   0              10y ago   \n",
		},
		{
			name: "acorn container kill found.container1", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"kill", "found.container1"},
				client: mClient,
			},
			wantErr: false,
			wantOut: "found.container1\n",
		},
		{
			name: "acorn container kill dne", fields: fields{
				All:    false,
				Quiet:  false,
				Output: "",
			},
			commandContext: CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader("y\n"),
			},
			args: args{
				args:   []string{"kill", "dne"},
				client: mClient,
			},
			wantErr: true,
			wantOut: "deleting dne: error: No such container: dne",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			tt.args.cmd = NewContainer(tt.commandContext)
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
