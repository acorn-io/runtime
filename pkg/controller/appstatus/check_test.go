package appstatus

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/condition"
	"github.com/acorn-io/runtime/pkg/scheme"
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

	condition.Setter(appInstance, resp, v1.AppInstanceConditionResolvedOfferings).Success()
	condition.Setter(appInstance, resp, v1.AppInstanceConditionScheduling).Success()
	condition.Setter(appInstance, resp, v1.AppInstanceConditionQuota).Success()

	if err := CheckStatus(handler).Handle(req, resp); err != nil {
		t.Fatal(err)
	}

	assert.True(t, called, "router handler call expected")
}
