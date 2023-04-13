package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/pkg/client"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/acorn-io/acorn/pkg/mocks"
	"github.com/golang/mock/gomock"
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

func TestRun(t *testing.T) {
	baseMock := func(f *mocks.MockClient) {
		f.EXPECT().AppGet(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, name string) (*apiv1.App, error) {
				switch name {
				case "dne":
					return nil, fmt.Errorf("error: app %s does not exist", name)
				case "found":
					return &apiv1.App{
						TypeMeta:   metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{Name: "found"},
						Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{{Secret: "found.secret", Target: "found"}}},
						Status:     v1.AppInstanceStatus{Ready: true},
					}, nil
				case "found.container":
					return &apiv1.App{
						TypeMeta:   metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
						Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{{Secret: "found.secret", Target: "found"}}},
						Status:     v1.AppInstanceStatus{},
					}, nil
				}
				return nil, nil
			}).AnyTimes()
		f.EXPECT().AppRun(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, image string, opts *client.AppRunOptions) (*apiv1.App, error) {
				switch image {
				case "dne":
					return nil, fmt.Errorf("error: app %s does not exist", image)
				case "found":
					return &apiv1.App{
						TypeMeta:   metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{Name: "found"},
						Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{{Secret: "found.secret", Target: "found"}}},
						Status:     v1.AppInstanceStatus{Ready: true},
					}, nil
				case "found.container":
					return &apiv1.App{
						TypeMeta:   metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
						Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{{Secret: "found.secret", Target: "found"}}},
						Status:     v1.AppInstanceStatus{},
					}, nil
				}
				return nil, fmt.Errorf("error: app %s does not exist", image)
			}).AnyTimes()
	}

	type fields struct {
		Quiet  bool
		Output string
		All    bool
		Force  bool
	}
	type args struct {
		cmd  *cobra.Command
		args []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantOut string
		prepare func(t *testing.T, f *mocks.MockClient)
	}{
		{
			name: "acorn run -h", fields: fields{
				All:   false,
				Force: true,
			},

			args: args{
				args: []string{"-h"},
			},
			wantErr: false,
			wantOut: "./testdata/run/acorn_run_help.txt",
		},
		{
			name: "acorn run -m found.container=256Miii found ", fields: fields{
				All:   false,
				Force: true,
			},
			args: args{
				args: []string{"-m found.container=256Miii", "found"},
			},
			wantErr: true,
			wantOut: "invalid number \"256Miii\"",
		},
		{
			name: "acorn run -m found.container=notallowed found ", fields: fields{
				All:   false,
				Force: true,
			},

			args: args{
				args: []string{"-m found.container=notallowed", "found"},
			},
			wantErr: true,
			wantOut: "illegal number start \"notallowed\"",
		},
		{
			name: "acorn run ./folder but folder doesn't exist", fields: fields{
				All:   false,
				Force: true,
			},

			args: args{
				args: []string{"./folder"},
			},
			prepare: func(t *testing.T, f *mocks.MockClient) {
				t.Helper()
				f.EXPECT().Info(gomock.Any()).Return(
					[]apiv1.Info{
						{
							TypeMeta:   metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{},
							Spec:       apiv1.InfoSpec{},
						},
					}, nil)
			},
			wantErr: true,
			wantOut: "directory ./folder does not exist",
		},
		{
			name: "acorn_run_pointed_at_working_dir_without_acornfile", fields: fields{
				All:   false,
				Force: true,
			},

			args: args{
				args: []string{"."},
			},
			prepare: func(t *testing.T, f *mocks.MockClient) {
				t.Helper()
				f.EXPECT().Info(gomock.Any()).Return(
					[]apiv1.Info{
						{
							TypeMeta:   metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{},
							Spec:       apiv1.InfoSpec{},
						},
					}, nil)
			},
			wantErr: true,
			wantOut: "open Acornfile: no such file or directory",
		},
		{
			name: "acorn_run_points_at_file", fields: fields{
				All:   false,
				Force: true,
			},
			args: args{
				args: []string{"Acornfile_temp"},
			},
			prepare: func(t *testing.T, f *mocks.MockClient) {
				t.Helper()
				// Create a placeholder acorn file
				dir, err := os.Getwd()
				if err != nil {
					t.Fatalf("failed to get current working directory: %v", err)
				}
				file, err := os.Create(dir + "/Acornfile_temp")
				if err != nil {
					t.Fatalf("failed to get current working directory: %v", err)
				}
				t.Cleanup(func() { os.Remove(file.Name()) })

				if _, err = file.Write([]byte("content")); err != nil {
					t.Fatal(err.Error())
				}
				if err = file.Close(); err != nil {
					t.Fatal(err)
				}
				if err != nil {
					t.Fatal()
				}

				f.EXPECT().Info(gomock.Any()).Return(
					[]apiv1.Info{
						{
							TypeMeta:   metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{},
							Spec:       apiv1.InfoSpec{},
						},
					}, nil)
			},
			wantErr: true,
			wantOut: "Acornfile_temp is not a directory",
		},
		{
			name: "acorn run --update --name dne", fields: fields{
				All:   false,
				Force: true,
			},
			args: args{
				args: []string{"--update", "--name", "dne"},
			},
			wantErr: true,
			wantOut: "error: app dne does not exist",
			prepare: func(t *testing.T, f *mocks.MockClient) {
				t.Helper()
				f.EXPECT().Info(gomock.Any()).Return(
					[]apiv1.Info{
						{
							TypeMeta:   metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{},
							Spec:       apiv1.InfoSpec{},
						},
					}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			//Mocked client for cli's client calls.
			mClient := mocks.NewMockClient(ctrl)

			baseMock(mClient)
			if tt.prepare != nil {
				tt.prepare(t, mClient)
			}

			r, w, _ := os.Pipe()
			os.Stdout = w
			// Mock client factory just returns the gomock client.
			tt.args.cmd = NewRun(CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader(""),
			})
			tt.args.cmd.SetArgs(tt.args.args)
			err := tt.args.cmd.Execute()

			if err != nil && !tt.wantErr {
				assert.Failf(t, "got err when err not expected", "got err: %s", err.Error())
			} else if err != nil && tt.wantErr {
				assert.Equal(t, tt.wantOut, err.Error())
			} else {
				w.Close()
				out, _ := io.ReadAll(r)
				wantOut := tt.wantOut
				if testOut, err := os.ReadFile(tt.wantOut); err == nil {
					wantOut = string(testOut)
				}
				assert.Equal(t, wantOut, string(out))
			}
		})
	}
}
