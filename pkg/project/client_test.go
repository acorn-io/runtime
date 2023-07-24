package project

import (
	"testing"
)

func TestParseProject(t *testing.T) {
	type args struct {
		project        string
		defaultContext string
	}
	tests := []struct {
		name          string
		args          args
		wantServer    string
		wantAccount   string
		wantNamespace string
		wantErr       bool
	}{
		{
			name: "Local namespace",
			args: args{
				project: "foo",
			},
			wantServer:    "kubeconfig",
			wantNamespace: "foo",
		},
		{
			name: "Manager reference",
			args: args{
				project: "manager.example.com/account/project",
			},
			wantNamespace: "project",
			wantAccount:   "account",
			wantServer:    "manager.example.com",
		},
		{
			name: "Manager reference",
			args: args{
				project: "manager.example.com/account/project",
			},
			wantNamespace: "project",
			wantAccount:   "account",
			wantServer:    "manager.example.com",
		},
		{
			name: "Manager default reference",
			args: args{
				project:        "account/project",
				defaultContext: "acorn.io/account",
			},
			wantNamespace: "project",
			wantAccount:   "account",
			wantServer:    "acorn.io",
		},
		{
			name: "Invalid server reference",
			args: args{
				project: "example.com/project",
			},
			wantErr: true,
		},
		{
			name: "Invalid length reference",
			args: args{
				project: "example.com/foo/bar/baz",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotServer, gotAccount, gotNamespace, _, err := ParseProject(tt.args.project, tt.args.defaultContext)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotServer != tt.wantServer {
				t.Errorf("ParseProject() gotServer = %v, want %v", gotServer, tt.wantServer)
			}
			if gotAccount != tt.wantAccount {
				t.Errorf("ParseProject() gotAccount = %v, want %v", gotAccount, tt.wantAccount)
			}
			if gotNamespace != tt.wantNamespace {
				t.Errorf("ParseProject() gotNamespace = %v, want %v", gotNamespace, tt.wantNamespace)
			}
		})
	}
}
