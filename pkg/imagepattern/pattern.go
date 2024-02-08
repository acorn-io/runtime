package imagepattern

import (
	"fmt"
	"regexp"
	"strings"
)

func IsImagePattern(image string) bool {
	return strings.ContainsAny(image, "#*")
}

// NamedMatchingGroup represents the information we need to know two things about a matching group: it's name and
// whether it should be sorted alphabetically or numerically. pType will be either "alpha" or "numeric"
type NamedMatchingGroup struct {
	PType string
	Name  string
}

// NewMatcher returns a Regexp that can be used to match an image or image tag against a pattern.
// The supplied pattern is NOT a regexp. It is acorn's own custom syntax with the following characteristics:
// - Assumed to be valid docker tag characters: 0-9A-Za-z_.-
// - Outside the special matching/sorting groups, a tag must match the pattern exactly
// - There are three special matching/sorting groups: #, *, and **
// - ** indicates a portion of the tag doesn't need to match the pattern and won't be considered for sorting. It is the "wildcard"
// - * indicates a portion of the tag that will be matched and sorted alphabetically
// - # indicates a portion of the tag that will be matched and sorted numerically
//
// Here are a few simple examples of patterns and what they would match:
// - "v#.#" - Matches: "v1.0", "v2.0" (return as latest). Doesn't match: "v1.alpha", "1.0", "v1.0.0"
// - "v1.0-*" - Matches: "v1.0-alpha", "v1.0-beta" (returned as latest). Doesn't match: "v1.0"
// - "v1.#-**" - Matches: "v1.0-cv23jkha", "v1.1-2020-01-01" (returned as latest).
func NewMatcher(pattern string) (*regexp.Regexp, []NamedMatchingGroup, error) {
	pattern = "^" + pattern + "$"

	// ** denotes a part of the tag that should be completely ignored for both matching and sorting. Replace it with
	// a regexp expression that matches all valid tag characters (and / and :)
	pattern = strings.ReplaceAll(pattern, "**", `([0-9A-Za-z_./:-]{0,})`)

	index := 0
	var namedMatchingGroups []NamedMatchingGroup

	// We are replacing the special cases of "#" and "*" with regex "Named Capturing Groups". We are using this feature
	// so that later we can sort on each group to find the "latest" image.
	// # denotes a part of the tag that should be parsed and sorted numerically.
	// * denotes a part of the tag that should be parsed and sorted alphabetically.
	// We are doing this in a loop and creating the namedMatchingGroups slice as we go so that the slice will represent
	// the groups as they appear from left-to-right in the tag. The left most group has the most precedence and it
	// decreases from there
	for strings.Contains(pattern, "#") || strings.Contains(pattern, "*") {
		name := fmt.Sprintf("m%v", index)

		if strings.Contains(pattern, "*") && (!strings.Contains(pattern, "#") || (strings.Index(pattern, "*") < strings.Index(pattern, "#"))) {
			pattern = strings.Replace(pattern, "*", fmt.Sprintf(`(?P<%v>[0-9A-Za-z_.-]+)`, name), 1)
			namedMatchingGroups = append(namedMatchingGroups, NamedMatchingGroup{PType: "alpha", Name: name})
		} else {
			pattern = strings.Replace(pattern, "#", fmt.Sprintf(`(?P<%v>\d+)`, name), 1)
			namedMatchingGroups = append(namedMatchingGroups, NamedMatchingGroup{PType: "numeric", Name: name})
		}

		index++
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return re, nil, err
	}

	return re, namedMatchingGroups, nil
}
