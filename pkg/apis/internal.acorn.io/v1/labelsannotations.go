package v1

import (
	"fmt"
	"strings"
)

var specialTypes = map[string]string{
	"container":  "container",
	"containers": "container",
	"job":        "job",
	"jobs":       "job",
	"volume":     "volume",
	"volumes":    "volume",
	"secret":     "secret",
	"secrets":    "secret",
	"app":        "app",
	// TODO - Figure out nested support
	"acorn":  "acorn",
	"acorns": "acorn",
}

func ParseScopedLabels(s ...string) (result []ScopedLabel, err error) {
	for _, s := range s {
		k, v, _ := strings.Cut(s, "=")
		scopeAndKeyParts := strings.Split(k, ":")
		var key, resourceType, resourceName string

		switch len(scopeAndKeyParts) {
		case 1:
			key = scopeAndKeyParts[0]
		case 2:
			scopePart := specialTypes[scopeAndKeyParts[0]]
			if scopePart != "" {
				resourceType = scopePart
			} else {
				resourceName = scopePart
			}
			key = scopeAndKeyParts[1]
		case 3:
			scopeTypePart := specialTypes[scopeAndKeyParts[0]]
			if scopeTypePart != "" {
				resourceType = scopeTypePart
			} else {
				return nil, fmt.Errorf("cannot parse label %v. Unrecognized scope type [%v]", k, scopeAndKeyParts[0])
			}

			resourceName = scopeAndKeyParts[1]
			key = scopeAndKeyParts[2]
		default:
			return nil, fmt.Errorf("cannot parse label %v. Unrecognized scope format", k)
		}

		result = append(result, ScopedLabel{
			ResourceType: resourceType,
			ResourceName: resourceName,
			Key:          key,
			Value:        v,
		})
	}
	return result, nil
}
