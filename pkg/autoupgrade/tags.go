package autoupgrade

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/acorn-io/runtime/pkg/imagepattern"
	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/sirupsen/logrus"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const invalidTag = "+notfound+"

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

func findLatestTagForImageWithPattern(ctx context.Context, c daemonClient, current, namespace, image, pattern string) (string, bool, error) {
	ref, tags, err := getTagsForImagePattern(ctx, c, namespace, strings.TrimSuffix(image, ":"+pattern))
	if err != nil {
		return "", false, err
	}

	newTag := current
	if newTag == "" {
		newTag = invalidTag
	}
	newImage := strings.TrimPrefix(ref.Context().Tag(newTag).Name(), defaultNoReg+"/")
	for len(tags) > 0 {
		nTag, err := FindLatest(newTag, pattern, tags)
		if err != nil || nTag == newTag {
			// resorting to current tag, so stop trying (we don't want to loop forever)
			break
		}
		img := strings.TrimPrefix(ref.Context().Tag(nTag).Name(), defaultNoReg+"/")
		if err := c.checkImageAllowed(ctx, namespace, img); err != nil {
			// remove the tag from the list and try again
			tags = slices.Filter(nil, tags, func(tag string) bool { return tag != nTag })
		} else {
			// found a valid tag that is allowed by all rules, so use it
			newTag = nTag
			newImage = img
			break
		}
	}

	// no new image needs to be returned since no new tags were found
	if newTag == invalidTag {
		return "", false, err
	}

	return newImage, newTag != current, err
}

// FindLatestTagForImageWithPattern will return the latest tag for image corresponding to the pattern.
func FindLatestTagForImageWithPattern(ctx context.Context, c kclient.Client, current, namespace, image, pattern string) (string, bool, error) {
	return findLatestTagForImageWithPattern(ctx, &client{c}, current, namespace, image, pattern)
}

// FindLatest returns the tag from the tags slice that sorts as the "latest" according to the supplied pattern.
func FindLatest(current, pattern string, tags []string) (string, error) {
	re, namedMatchingGroups, err := imagepattern.NewMatcher(pattern)
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
				index := re.SubexpIndex(p.Name)
				if p.PType == "alpha" {
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
