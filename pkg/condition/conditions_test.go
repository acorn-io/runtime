package condition

import (
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_reflector_Conditions(t *testing.T) {
	v := &v1.ServiceInstance{}
	c := ForName(v, "foo")
	c.Unknown("bar")

	assert.Equal(t, "foo", v.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionStatus("Unknown"), v.Status.Conditions[0].Status)
	assert.Equal(t, "InProgress", v.Status.Conditions[0].Reason)
	assert.Equal(t, "bar", v.Status.Conditions[0].Message)
}
