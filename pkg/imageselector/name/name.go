package name

import (
	"fmt"
	"strings"

	"github.com/acorn-io/runtime/pkg/imagepattern"
	"github.com/google/go-containerregistry/pkg/name"
)

func ImageCovered(image name.Reference, digest string, patterns []string) bool {
	for _, pattern := range patterns {
		// empty pattern? skip (should've been caught by IAR validation already)
		if strings.TrimSpace(pattern) == "" {
			continue
		}

		// not a pattern? must be exact match then.
		if !imagepattern.IsImagePattern(pattern) {
			if strings.TrimSuffix(image.Name(), ":") != pattern && digest != pattern {
				continue
			}
			return true
		}

		tagPattern := ""
		contextPattern := ""

		if strings.Contains(pattern, "@") {
			parts := strings.Split(pattern, "@")
			contextPattern = parts[0]
			tagPattern = parts[1]
		} else {
			parts := strings.Split(pattern, ":")
			contextPattern = parts[0]
			if len(parts) > 1 {
				if !strings.Contains(parts[len(parts)-1], "/") {
					tagPattern = parts[len(parts)-1] // last part is tag
					contextPattern = strings.TrimSuffix(pattern, ":"+tagPattern)
				} else {
					contextPattern = pattern // : was part of the context pattern (port)
				}
			}
		}

		if err := matchContext(contextPattern, image.Context().String()); err != nil {
			continue
		}

		if tagPattern != "" {
			if err := matchTag(tagPattern, image.Identifier()); err != nil {
				continue
			}
		}

		return true
	}
	return false
}

// matchContext matches the image context against the context pattern, similar to globbing
func matchContext(contextPattern string, imageContext string) error {
	re, _, err := imagepattern.NewMatcher(contextPattern)
	if err != nil {
		return fmt.Errorf("error parsing context pattern %s: %w", contextPattern, err)
	}

	if re.MatchString(imageContext) {
		return nil
	}

	return fmt.Errorf("image context %s does not match pattern %s (regex: `%s`)", imageContext, contextPattern, re.String())
}

// matchTag matches the image tag against the tag pattern, similar to auto-upgrade pattern
func matchTag(tagPattern string, imageTag string) error {
	re, _, err := imagepattern.NewMatcher(tagPattern)
	if err != nil {
		return fmt.Errorf("error parsing tag pattern %s: %w", tagPattern, err)
	}

	if re.MatchString(imageTag) {
		return nil
	}

	return fmt.Errorf("image tag %s does not match pattern %s (regex: `%s`)", imageTag, tagPattern, re.String())
}
