package v1

import (
	"fmt"
	"strings"
)

const (
	LabelTypeContainer = "container"
	LabelTypeJob       = "job"
	LabelTypeVolume    = "volume"
	LabelTypeSecret    = "secret"
	LabelTypeMeta      = "metadata"
	LabelTypeAcorn     = "acorn"
)

var specialTypes = map[string]string{
	"container":  LabelTypeContainer,
	"containers": LabelTypeContainer,
	"job":        LabelTypeJob,
	"jobs":       LabelTypeJob,
	"volume":     LabelTypeVolume,
	"volumes":    LabelTypeVolume,
	"secret":     LabelTypeSecret,
	"secrets":    LabelTypeSecret,
	"metadata":   LabelTypeMeta,
	"acorn":      LabelTypeAcorn,
	"acorns":     LabelTypeAcorn,
	// TODO - Figure out nested support
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
				resourceName = scopeAndKeyParts[0]
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
			if resourceName == "" {
				return nil, fmt.Errorf("cannot parse label %v. Unrecognized scope format", k)
			}

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
