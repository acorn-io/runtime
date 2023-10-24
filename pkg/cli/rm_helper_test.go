package cli

import (
	"testing"

	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/cli/testdata"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSecretsToRemove(t *testing.T) {
	client := &testdata.MockClient{}
	client.Secrets = []v1.Secret{
		{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
			Type:       "",
			Data:       nil,
			Keys:       nil,
		},
		{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.secret.nested"},
			Type:       "",
			Data:       nil,
			Keys:       nil,
		},
	}
	cmd := &cobra.Command{}
	tests := []struct {
		name     string
		arg      string
		expected []string
	}{
		{
			"simple",
			"found",
			[]string{"found.secret"},
		},
		{
			"nested",
			"found.secret",
			[]string{"found.secret.nested"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets, err := getSecretsToRemove(tt.arg, client, cmd)
			assert.NoError(t, err)
			assert.EqualValues(t, tt.expected, secrets)
		})
	}
}
