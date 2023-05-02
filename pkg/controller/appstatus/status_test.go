package appstatus

import (
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
)

func TestCheckStatus(t *testing.T) {
	appInstance := new(v1.AppInstance)
	req := tester.NewRequest(t, scheme.Scheme, appInstance)
	resp := &tester.Response{Client: req.Client.(*tester.Client)}
	called := false

	handler := router.HandlerFunc(func(router.Request, router.Response) error {
		called = true
		return nil
	})

	if err := CheckStatus(handler).Handle(req, resp); err != nil {
		t.Fatal(err)
	}
	assert.False(t, called, "router handler call unexpected")

	condition.Setter(appInstance, resp, v1.AppInstanceConditionDefaults).Success()
	condition.Setter(appInstance, resp, v1.AppInstanceConditionScheduling).Success()

	if err := CheckStatus(handler).Handle(req, resp); err != nil {
		t.Fatal(err)
	}

	assert.True(t, called, "router handler call expected")
}
