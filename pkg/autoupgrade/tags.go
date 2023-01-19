package autoupgrade

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/sirupsen/logrus"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getTagsForImagePattern(ctx context.Context, c daemonClient, namespace, image string) (imagename.Reference, []string, error) {
	current, err := imagename.ParseReference(image, imagename.WithDefaultRegistry(defaultNoReg))
	if err != nil {
		return nil, nil, fmt.Errorf("problem parsing image referece %v: %v", image, err)
	}
	// if the registry after being parsed is our default fake one, then this is a local image with no registry
	hasValidRegistry := current.Context().RegistryStr() != defaultNoReg
	var tags []string
	var pullErr error
	if hasValidRegistry {
		tags, pullErr = c.listTags(ctx, namespace, image)
	}
	localTags, err := c.getTagsMatchingRepo(ctx, current, namespace, defaultNoReg)
	if err != nil {
		logrus.Errorf("Problem finding local tags matching %v: %v", image, err)
	}
	if len(localTags) == 0 && pullErr != nil {
		logrus.Errorf("Couldn't find any remote tags for image %v. Error: %v", image, pullErr)
	}
	return current, append(tags, localTags...), nil
}

func findLatestTagForImageWithPattern(ctx context.Context, c daemonClient, namespace, image, pattern string) (string, bool, error) {
	ref, tags, err := getTagsForImagePattern(ctx, c, namespace, strings.TrimSuffix(image, ":"+pattern))
	if err != nil {
		return "", false, err
	}

	newTag, err := FindLatest(imagename.DefaultTag, pattern, tags)
	return strings.TrimPrefix(ref.Context().Tag(newTag).Name(), defaultNoReg+"/"), newTag != imagename.DefaultTag, err
}

// FindLatestTagForImageWithPattern will return the latest tag for image corresponding to the pattern.
func FindLatestTagForImageWithPattern(ctx context.Context, c kclient.Client, namespace, image, pattern string) (string, bool, error) {
	return findLatestTagForImageWithPattern(ctx, &client{c}, namespace, image, pattern)
}

// FindLatest returns the tag from the tags slice that sorts as the "latest" according to the supplied pattern. The supplied
// pattern is NOT a regex. It is acorn's own custom syntax with the following characteristics:
// - Assumed to be valid docker tag characters: 0-9A-Za-z_.-
// - Outside of the special matching/sorting groups, a tag must match the pattern exactly
// - There are three special matching/sorting groups: #, *, and **
// - ** indicates a portion of the tag doesn't need to match the pattern and won't be considered for sorting. It is the "wildcard"
// - * indicates a portion of the tag that will be matched and sorted alphabetically
// - # indicates a portion of the tag that will be matched and sorted numerically
//
// Here are a few simple examples of patterns and what they would match:
// - "v#.#" - Matches: "v1.0", "v2.0" (return as latest). Doesn't match: "v1.alpha", "1.0", "v1.0.0"
// - "v1.0-*" - Matches: "v1.0-alpha", "v1.0-beta" (returned as latest). Doesn't match: "v1.0"
// - "v1.#-**" - Matches: "v1.0-cv23jkha", "v1.1-2020-01-01" (returned as latest).
func FindLatest(current, pattern string, tags []string) (string, error) {
	pattern = "^" + pattern + "$"

	// ** denotes a part of the tag that should be completely ignored for both matching and sorting. Replace it with
	// a regexp expression that matches all valid tag characters
	pattern = strings.ReplaceAll(pattern, "**", `([0-9A-Za-z_.-]+)`)

	index := 0
	var namedMatchingGroups []namedMatchingGroup

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
			namedMatchingGroups = append(namedMatchingGroups, namedMatchingGroup{pType: "alpha", name: name})
		} else {
			pattern = strings.Replace(pattern, "#", fmt.Sprintf(`(?P<%v>\d+)`, name), 1)
			namedMatchingGroups = append(namedMatchingGroups, namedMatchingGroup{pType: "numeric", name: name})
		}

		index += 1
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	latest := current
	var latestMatches []string
	if re.MatchString(latest) {
		matches := re.FindStringSubmatch(latest)
		latestMatches = slices.Clone(matches)
	}
	// This is the logic that will select the "latest" tag to match the given pattern.
	// If a tag matches the pattern, we then get a slice of the "submatches", which correspond to the named capturing
	// groups from above.
	for _, tag := range tags {
		if re.MatchString(tag) {
			matches := re.FindStringSubmatch(tag)

			if len(latestMatches) == 0 {
				// We are here because the "current" tag didn't match the pattern. In this case, we assume that any tag
				// that matches the pattern is "later" than current and should replace it
				latest = tag
				latestMatches = slices.Clone(matches)
				continue
			}

			// Find the value for each namedMatchingGroup, if it sorts as greater than latest's value for the same
			// matching group, then this tag becomes the new "latest" tag. Set it as latest and break out of this
			// inner loop to continue on to the next tag
			for _, p := range namedMatchingGroups {
				index := re.SubexpIndex(p.name)
				if p.pType == "alpha" {
					// Type is alphabetical
					if matches[index] < latestMatches[index] {
						break
					} else if matches[index] > latestMatches[index] {
						latest = tag
						copy(latestMatches, matches)
						break
					}
				} else {
					// Type is numeric, need to convert to ints and compare
					latestInt, err := strconv.Atoi(latestMatches[index])
					if err != nil {
						return "", err
					}
					matchInt, err := strconv.Atoi(matches[index])
					if err != nil {
						return "", err
					}
					if matchInt < latestInt {
						break
					} else if matchInt > latestInt {
						latest = tag
						copy(latestMatches, matches)
						break
					}
				}
			}
		}
	}

	return latest, nil
}

// We need to know two things about a matching group: it's name and whether it should be sorted alphabetically or
// numerically. pType will be either "alpha" or "numeric"
type namedMatchingGroup struct {
	pType string
	name  string
}
