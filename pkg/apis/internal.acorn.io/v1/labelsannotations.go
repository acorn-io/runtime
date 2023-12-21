package v1

import (
	"fmt"
	"strings"
)

const (
	LabelTypeRouter    = "router"
	LabelTypeContainer = "container"
	LabelTypeFunction  = "function"
	LabelTypeJob       = "job"
	LabelTypeVolume    = "volume"
	LabelTypeSecret    = "secret"
	LabelTypeMeta      = "metadata"
	LabelTypeAcorn     = "acorn"
	LabelTypeService   = "service"
)

var canonicalTypes = map[string]string{
	"router":     LabelTypeRouter,
	"routers":    LabelTypeRouter,
	"container":  LabelTypeContainer,
	"containers": LabelTypeContainer,
	"function":   LabelTypeFunction,
	"functions":  LabelTypeFunction,
	"job":        LabelTypeJob,
	"jobs":       LabelTypeJob,
	"volume":     LabelTypeVolume,
	"volumes":    LabelTypeVolume,
	"secret":     LabelTypeSecret,
	"secrets":    LabelTypeSecret,
	"metadata":   LabelTypeMeta,
	"metadatas":  LabelTypeMeta,
	"acorn":      LabelTypeAcorn,
	"acorns":     LabelTypeAcorn,
	"service":    LabelTypeService,
	"services":   LabelTypeService,
}

// ParseScopedLabels parses labels from their string format into the struct form. Examples of the string format:
// --label k=v (global, no resource scope)
// --label containers:k=v (apply to all containers)
// --label containers:foo:k=v (apply to container named foo)
// --label foo:k=v (apply to any resource named foo)
func ParseScopedLabels(s ...string) (result []ScopedLabel, err error) {
	for _, s := range s {
		k, v, _ := strings.Cut(s, "=")
		l, err := parseScopedLabel(k, v)
		if err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, nil
}

// similar to the above function, but the string has been split into a key and value already, like:
// key="containers:foo:k" value="v"
func parseScopedLabel(k, v string) (ScopedLabel, error) {
	scopeAndKeyParts := strings.Split(k, ":")
	var key, resourceType, resourceName string

	switch len(scopeAndKeyParts) {
	case 1:
		key = scopeAndKeyParts[0]
	case 2:
		scopePart := canonicalTypes[strings.ToLower(scopeAndKeyParts[0])]
		if scopePart != "" {
			resourceType = scopePart
		} else {
			resourceName = scopeAndKeyParts[0]
		}
		key = scopeAndKeyParts[1]
	case 3:
		scopeTypePart := canonicalTypes[strings.ToLower(scopeAndKeyParts[0])]
		if scopeTypePart != "" {
			resourceType = scopeTypePart
		} else {
			return ScopedLabel{}, fmt.Errorf("cannot parse label %v. Unrecognized scope type [%v]", k, scopeAndKeyParts[0])
		}

		resourceName = scopeAndKeyParts[1]
		if resourceName == "" {
			return ScopedLabel{}, fmt.Errorf("cannot parse label %v. Unrecognized scope format", k)
		}

		key = scopeAndKeyParts[2]
	default:
		return ScopedLabel{}, fmt.Errorf("cannot parse label %v. Unrecognized scope format", k)
	}

	return ScopedLabel{
		ResourceType: resourceType,
		ResourceName: resourceName,
		Key:          key,
		Value:        v,
	}, nil
}

func canonicalResourceType(rType string) (string, error) {
	if rType == "" {
		return rType, nil
	}
	normalized := canonicalTypes[strings.ToLower(rType)]
	if normalized == "" {
		return "", fmt.Errorf("unrecognized scope resourceType [%v]", rType)
	}
	return normalized, nil
}
