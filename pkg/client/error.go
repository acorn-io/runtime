package client

import (
	"encoding/json"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var (
	PrefixErrRulesNeeded = "rules needed: "
)

type ErrRulesNeeded struct {
	Permissions v1.Permissions
}

func (e *ErrRulesNeeded) Error() string {
	perms, err := json.Marshal(e.Permissions)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s%s", PrefixErrRulesNeeded, perms)
}

type ErrNotAuthorized struct {
	Rule rbacv1.PolicyRule
}

func (e *ErrNotAuthorized) Error() string {
	perms, err := json.Marshal(e.Rule)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("not authorized: %s", perms)
}
