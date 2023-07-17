package init

import "github.com/acorn-io/runtime/pkg/logserver"

func init() {
	go logserver.StartServerWithDefaults()
}
