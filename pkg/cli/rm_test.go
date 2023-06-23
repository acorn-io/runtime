package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	api_acorn_io "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/cli/testdata"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/mocks"
	"github.com/acorn-io/runtime/pkg/prompt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestAppRm(t *testing.T) {
	type fields struct {
		All   bool
		Type  []string
		Force bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    []string
		prepare func(client *mocks.MockClient)
		stdin   string
		wantErr bool
		wantOut string
	}{
		{
			name: "does not exist default type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().AppDelete(gomock.Any(), "dne").Return(
					nil, fmt.Errorf("error: app dne does not exist"),
				)
			},
			stdin:   "y\n",
			args:    []string{"dne"},
			wantErr: true,
			wantOut: "deleting app dne: error: app dne does not exist",
		},
		{
			name: "does not exist force default type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: true,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().AppDelete(gomock.Any(), "dne").Return(
					nil, fmt.Errorf("error: app dne does not exist"),
				)
			},
			args:    []string{"-f", "dne"},
			wantErr: true,
			wantOut: "deleting app dne: error: app dne does not exist",
		},
		{
			name: "does not exist container type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ContainerReplicaList(gomock.Any(), nil).Return(
					nil, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-tc", "dne"},
			wantErr: false,
			wantOut: "",
		},
		{
			name: "does not exist volume type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().VolumeList(gomock.Any()).Return(
					nil, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-tv", "dne"},
			wantErr: false,
			wantOut: "",
		},
		{
			name: "does exist default type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().AppDelete(gomock.Any(), "found").Return(
					&apiv1.App{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found",
						},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"found"},
			wantErr: false,
			wantOut: "Removed: found\n",
		},
		{
			name: "does exist force default type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: true,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().AppDelete(gomock.Any(), "found").Return(
					&apiv1.App{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found",
						},
					}, nil,
				)
			},
			args:    []string{"-f", "found"},
			wantErr: false,
			wantOut: "Removed: found\n",
		},
		{
			name: "does exist container type short",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ContainerReplicaList(gomock.Any(), nil).Return(
					[]apiv1.ContainerReplica{{
						ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
						Spec:       apiv1.ContainerReplicaSpec{AppName: "found"},
					}}, nil,
				)
				f.EXPECT().ContainerReplicaDelete(gomock.Any(), "found.container").Return(
					&apiv1.ContainerReplica{
						ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
						Spec:       apiv1.ContainerReplicaSpec{AppName: "found"},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-tc", "found"},
			wantErr: false,
			wantOut: "Removed: found.container\n",
		},
		{
			name: "does exist container type long",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ContainerReplicaList(gomock.Any(), nil).Return(
					[]apiv1.ContainerReplica{{
						ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
						Spec:       apiv1.ContainerReplicaSpec{AppName: "found"},
					}}, nil,
				)
				f.EXPECT().ContainerReplicaDelete(gomock.Any(), "found.container").Return(
					&apiv1.ContainerReplica{
						ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
						Spec:       apiv1.ContainerReplicaSpec{AppName: "found"},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-tcontainer", "found"},
			wantErr: false,
			wantOut: "Removed: found.container\n",
		},
		{
			name: "does exist app type short",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().AppDelete(gomock.Any(), "found").Return(
					&apiv1.App{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found",
						},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-ta", "found"},
			wantErr: false,
			wantOut: "Removed: found\n",
		},
		{
			name: "does exist app type long",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().AppDelete(gomock.Any(), "found").Return(
					&apiv1.App{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found",
						},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-tapp", "found"},
			wantErr: false,
			wantOut: "Removed: found\n",
		},
		{
			name: "does exist volume type short",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().VolumeList(gomock.Any()).Return(
					[]apiv1.Volume{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found.vol",
							Labels: map[string]string{
								labels.AcornVolumeName: "vol",
								labels.AcornAppName:    "found",
							}},
						Status: apiv1.VolumeStatus{
							AppPublicName: "found",
							AppName:       "found",
							VolumeName:    "vol",
						},
					}}, nil,
				)
				f.EXPECT().VolumeDelete(gomock.Any(), "found.vol").Return(
					&apiv1.Volume{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found.vol",
						},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-tv", "found"},
			wantErr: false,
			wantOut: "Removed: found.vol\n",
		},
		{
			name: "does exist volume type long",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().VolumeList(gomock.Any()).Return(
					[]apiv1.Volume{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found.vol",
							Labels: map[string]string{
								labels.AcornVolumeName: "vol",
								labels.AcornAppName:    "found",
							}},
						Status: apiv1.VolumeStatus{
							AppPublicName: "found",
							AppName:       "found",
							VolumeName:    "vol",
						},
					}}, nil,
				)
				f.EXPECT().VolumeDelete(gomock.Any(), "found.vol").Return(
					&apiv1.Volume{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found.vol",
						},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-tvolume", "found"},
			wantErr: false,
			wantOut: "Removed: found.vol\n",
		},
		{
			name: "does exist secret type short",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().SecretList(gomock.Any()).Return(
					[]apiv1.Secret{{
						ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
					}}, nil,
				)
				f.EXPECT().SecretDelete(gomock.Any(), "found.secret").Return(
					&apiv1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-ts", "found"},
			wantErr: false,
			wantOut: "Removed: found.secret\n",
		},
		{
			name: "does exist secret type long",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().SecretList(gomock.Any()).Return(
					[]apiv1.Secret{{
						ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
					}}, nil,
				)
				f.EXPECT().SecretDelete(gomock.Any(), "found.secret").Return(
					&apiv1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-tsecret", "found"},
			wantErr: false,
			wantOut: "Removed: found.secret\n",
		},
		{
			name: "does exist all type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().AppDelete(gomock.Any(), "found").Return(
					&apiv1.App{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found",
						},
					}, nil,
				)
				gomock.InOrder(
					f.EXPECT().AppGet(gomock.Any(), "found").Return(
						&apiv1.App{
							ObjectMeta: metav1.ObjectMeta{
								Name: "found",
							},
						}, nil,
					),
					f.EXPECT().AppGet(gomock.Any(), "found").Return(
						nil, apierrors.NewNotFound(schema.GroupResource{
							Group:    api_acorn_io.Group,
							Resource: "apps",
						}, "found"),
					),
				)
				f.EXPECT().VolumeList(gomock.Any()).Return(
					[]apiv1.Volume{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found.vol",
							Labels: map[string]string{
								labels.AcornVolumeName: "vol",
								labels.AcornAppName:    "found",
							}},
						Status: apiv1.VolumeStatus{
							AppPublicName: "found",
							AppName:       "found",
							VolumeName:    "vol",
						},
					}}, nil,
				)
				f.EXPECT().VolumeDelete(gomock.Any(), "found.vol").Return(
					&apiv1.Volume{
						ObjectMeta: metav1.ObjectMeta{
							Name: "found.vol",
						},
					}, nil,
				)
				f.EXPECT().SecretList(gomock.Any()).Return(
					[]apiv1.Secret{{
						ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
					}}, nil,
				)
				f.EXPECT().SecretDelete(gomock.Any(), "found.secret").Return(
					&apiv1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
					}, nil,
				)
			},
			stdin:   "y\n",
			args:    []string{"-a", "found"},
			wantErr: false,
			wantOut: "Waiting for app found to be removed...\nRemoved: found\nRemoved: found.vol\nRemoved: found.secret\n",
		},
		{
			name: "no app name arg default type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: false,
			},
			stdin:   "y\n",
			args:    []string{},
			wantErr: true,
			wantOut: "requires at least 1 arg(s), only received 0",
		},
		{
			name: "no app name arg force default type",
			fields: fields{
				All:   false,
				Type:  nil,
				Force: true,
			},
			args:    []string{"-f"},
			wantErr: true,
			wantOut: "requires at least 1 arg(s), only received 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mClient := mocks.NewMockClient(ctrl)
			if tt.prepare != nil {
				tt.prepare(mClient)
			}

			r, w, _ := os.Pipe()
			os.Stdout = w
			cmd := NewRm(CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client: mClient,
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader(tt.stdin),
			})
			cmd.SetArgs(tt.args)
			prompt.NoPromptRemove = true

			err := cmd.Execute()
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
