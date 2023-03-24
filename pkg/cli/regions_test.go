package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRegions(t *testing.T) {
	tenYearsAgo := metav1.Now().AddDate(-10, 0, 0)
	tests := []struct {
		name            string
		existingRegions []apiv1.Region
		quiet           bool
		args            []string
		wantErr         bool
		wantOut         string
	}{
		{
			name: "acorn regions with one region",
			existingRegions: []apiv1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "local",
						CreationTimestamp: metav1.NewTime(tenYearsAgo),
					},
					Spec: v1.RegionInstanceSpec{
						Description: "Test region",
						AccountName: "my-account",
						Role:        "arn:aws:iam::123456789012:role/role-name",
						RegionName:  "us-east-2",
					},
				},
			},
			args:    []string{},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME      ACCOUNT      REGION NAME   CREATED   DESCRIPTION\nlocal     my-account   us-east-2     10y ago   Test region\n",
		},
		{
			name: "acorn regions with multiple regions",
			existingRegions: []apiv1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "local",
						CreationTimestamp: metav1.NewTime(tenYearsAgo),
					},
					Spec: v1.RegionInstanceSpec{
						Description: "Test region",
						AccountName: "my-account",
						Role:        "arn:aws:iam::123456789012:role/role-name",
						RegionName:  "us-east-2",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "local",
						CreationTimestamp: metav1.NewTime(tenYearsAgo),
					},
					Spec: v1.RegionInstanceSpec{
						Description: "Another test region",
						AccountName: "my-account",
						Role:        "arn:aws:iam::123456789012:role/role-name",
						RegionName:  "us-west-2",
					},
				},
			},
			args:    []string{},
			quiet:   false,
			wantErr: false,
			wantOut: "NAME      ACCOUNT      REGION NAME   CREATED   DESCRIPTION\nlocal     my-account   us-east-2     10y ago   Test region\nlocal     my-account   us-west-2     10y ago   Another test region\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			cmd := NewRegions(CommandContext{
				ClientFactory: &testdata.MockClientFactory{RegionList: tt.existingRegions},
				StdOut:        w,
				StdErr:        w,
				StdIn:         strings.NewReader(""),
			})
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err != nil && !tt.wantErr {
				assert.Failf(t, "got err when err not expected", "got err: %s", err.Error())
			} else if err != nil && tt.wantErr {
				assert.Equal(t, tt.wantOut, err.Error())
			} else {
				assert.Nil(t, w.Close(), "error closing writer")
				out, _ := io.ReadAll(r)
				assert.Equal(t, tt.wantOut, string(out))
			}
		})
	}
}
