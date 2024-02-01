package v1

import (
	"errors"
	"fmt"
	"strings"
)

// BaseResources defines resources that should be tracked at any scoped. The two main exclusions
// currently are Secrets and Projects as they have situations they should be not be tracked.
type BaseResources struct {
	Apps       int `json:"apps"`
	Containers int `json:"containers"`
	Jobs       int `json:"jobs"`
	Volumes    int `json:"volumes"`
	Images     int `json:"images"`

	// ComputeClasses and VolumeClasses are used to track the amount of compute and volume storage per their
	// respective classes
	ComputeClasses ComputeClassResources `json:"computeClasses"`
	VolumeClasses  VolumeClassResources  `json:"volumeClasses"`
}

// Add will add the BaseResources of another BaseResources struct into the current one.
func (current *BaseResources) Add(incoming BaseResources) {
	current.Apps = Add(current.Apps, incoming.Apps)
	current.Containers = Add(current.Containers, incoming.Containers)
	current.Jobs = Add(current.Jobs, incoming.Jobs)
	current.Volumes = Add(current.Volumes, incoming.Volumes)
	current.Images = Add(current.Images, incoming.Images)

	if current.ComputeClasses == nil {
		current.ComputeClasses = ComputeClassResources{}
	}
	if current.VolumeClasses == nil {
		current.VolumeClasses = VolumeClassResources{}
	}
	current.ComputeClasses.Add(incoming.ComputeClasses)
	current.VolumeClasses.Add(incoming.VolumeClasses)
}

// Remove will remove the BaseResources of another BaseResources struct from the current one. Calling remove
// will be a no-op for any resource values that are set to unlimited.
func (current *BaseResources) Remove(incoming BaseResources, all bool) {
	current.Apps = Sub(current.Apps, incoming.Apps)
	current.Containers = Sub(current.Containers, incoming.Containers)
	current.Jobs = Sub(current.Jobs, incoming.Jobs)
	current.Volumes = Sub(current.Volumes, incoming.Volumes)
	current.Images = Sub(current.Images, incoming.Images)
	current.ComputeClasses.Remove(incoming.ComputeClasses)
	if all {
		current.VolumeClasses.Remove(incoming.VolumeClasses)
	}
}

// Fits will check if a group of BaseResources will be able to contain
// another group of BaseResources. If the BaseResources are not able to fit,
// an aggregated error will be returned with all exceeded BaseResources.
// If the current BaseResources defines unlimited, then it will always fit.
func (current *BaseResources) Fits(incoming BaseResources) error {
	var exceededResources []string
	var errs []error

	// Check if any of the resources are exceeded
	for _, r := range []struct {
		resource          string
		current, incoming int
	}{
		{"Apps", current.Apps, incoming.Apps},
		{"Containers", current.Containers, incoming.Containers},
		{"Jobs", current.Jobs, incoming.Jobs},
		{"Volumes", current.Volumes, incoming.Volumes},
		{"Images", current.Images, incoming.Images},
	} {
		if !Fits(r.current, r.incoming) {
			exceededResources = append(exceededResources, r.resource)
		}
	}

	if len(exceededResources) != 0 {
		errs = append(errs, fmt.Errorf("%w: %s", ErrExceededResources, strings.Join(exceededResources, ", ")))
	}

	if err := current.ComputeClasses.Fits(incoming.ComputeClasses); err != nil {
		errs = append(errs, err)
	}

	if err := current.VolumeClasses.Fits(incoming.VolumeClasses); err != nil {
		errs = append(errs, err)
	}

	// Build an aggregated error message for the exceeded resources
	return errors.Join(errs...)
}

// ToString will return a string representation of the BaseResources within the struct.
func (current *BaseResources) ToString() string {
	// make sure that an empty string doesn't have a comma
	result := CountResourcesToString(
		map[string]int{
			"Apps":       current.Apps,
			"Containers": current.Containers,
			"Jobs":       current.Jobs,
			"Volumes":    current.Volumes,
			"Images":     current.Images,
		},
	)

	for _, resource := range []struct {
		name     string
		asString string
	}{
		{"ComputeClasses", current.ComputeClasses.ToString()},
		{"VolumeClasses", current.VolumeClasses.ToString()},
	} {
		if result != "" && resource.asString != "" {
			result += ", "
		}
		if resource.asString != "" {
			result += fmt.Sprintf("%s: %s", resource.name, resource.asString)
		}
	}

	return result
}

// Equals will check if the current BaseResources struct is equal to another. This is useful
// to avoid needing to do a deep equal on the entire struct.
func (current *BaseResources) Equals(incoming BaseResources) bool {
	return current.Apps == incoming.Apps &&
		current.Containers == incoming.Containers &&
		current.Jobs == incoming.Jobs &&
		current.Volumes == incoming.Volumes &&
		current.Images == incoming.Images &&
		current.ComputeClasses.Equals(incoming.ComputeClasses) &&
		current.VolumeClasses.Equals(incoming.VolumeClasses)
}
