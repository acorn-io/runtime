package client

import (
	"encoding/json"
	"fmt"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

var (
	PrefixErrRulesNeeded  = "rules needed: "
	PrefixErrRulesMissing = "rules missing: "
)

type ErrRulesNeeded struct {
	Missing     []v1.Permissions
	Permissions []v1.Permissions
}

func (e *ErrRulesNeeded) Error() string {
	prefix := PrefixErrRulesNeeded
	if len(e.Missing) > 0 {
		perms, err := json.Marshal(e.Permissions)
		if err != nil {
			panic(err)
		}
		prefix = PrefixErrRulesMissing + string(perms) + ", " + PrefixErrRulesNeeded
	}
	perms, err := json.Marshal(e.Permissions)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s%s", prefix, perms)
}

type ErrNotAuthorized struct {
	Permissions []v1.Permissions
}

func (e *ErrNotAuthorized) Error() string {
	perms, err := json.Marshal(e.Permissions)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("not authorized: %s", perms)
}

// ErrRegistryUnauthorized denotes that we're missing credentials for a registry
type ErrRegistryUnauthorized struct {
	Image string
}

func (e *ErrRegistryUnauthorized) Error() string {
	return fmt.Sprintf("not authorized to pull %s - Use `acorn login REGISTRY` to login to the registry", e.Image)
}
