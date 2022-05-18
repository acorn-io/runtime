package version

import (
	"fmt"
	"runtime/debug"
)

var (
	Tag string
)

func Version(tag string) string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return tag
	}
	var (
		dirty  bool
		commit string
	)

	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.modified":
			dirty = setting.Value == "true"
		case "vcs.revision":
			commit = setting.Value
		}
	}

	if len(commit) < 12 {
		return tag
	} else if dirty {
		return fmt.Sprintf("%s-%s-dirty", tag, commit[:8])
	}

	return fmt.Sprintf("%s+%s", tag, commit[:8])
}
