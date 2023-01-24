package imagesystem

import (
	"testing"
)

func Test_isNotInternalRepo(t *testing.T) {
	type args struct {
		prefix string
		image  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no prefix",
			args: args{
				prefix: "",
				image:  "foo/bar",
			},
		},
		{
			name: "simple prefix",
			args: args{
				prefix: "foo/ba",
				image:  "foo/bar",
			},
			wantErr: true,
		},
		{
			name: "normalize prefix",
			args: args{
				prefix: "docker.io:1234/ba",
				image:  "registry-1.docker.io:3434/bar",
			},
			wantErr: true,
		},
		{
			name: "normalize ports",
			args: args{
				prefix: "host:1234/ba",
				image:  "host:4444/bar",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := isNotInternalRepo(tt.args.prefix, tt.args.image); (err != nil) != tt.wantErr {
				t.Errorf("isNotInternalRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
