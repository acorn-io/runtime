package v1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

// BaseResources defines resources that should be tracked at any scoped. The two main exclusions
// currently are Secrets and Projects as they have situations they should be not be tracked.
type BaseResources struct {
	Apps       int `json:"apps"`
	Containers int `json:"containers"`
	Jobs       int `json:"jobs"`
	Volumes    int `json:"volumes"`
	Images     int `json:"images"`

	VolumeStorage resource.Quantity `json:"volumeStorage"`
	Memory        resource.Quantity `json:"memory"`
	CPU           resource.Quantity `json:"cpu"`
}

// Add will add the BaseResources of another BaseResources struct into the current one.
func (current *BaseResources) Add(incoming BaseResources) {
	current.Apps = Add(current.Apps, incoming.Apps)
	current.Containers = Add(current.Containers, incoming.Containers)
	current.Jobs = Add(current.Jobs, incoming.Jobs)
	current.Volumes = Add(current.Volumes, incoming.Volumes)
	current.Images = Add(current.Images, incoming.Images)

	current.VolumeStorage = AddQuantity(current.VolumeStorage, incoming.VolumeStorage)
	current.Memory = AddQuantity(current.Memory, incoming.Memory)
	current.CPU = AddQuantity(current.CPU, incoming.CPU)
}

// Remove will remove the BaseResources of another BaseResources struct from the current one. Calling remove
// will be a no-op for any resource values that are set to unlimited.
func (current *BaseResources) Remove(incoming BaseResources, all bool) {
	current.Apps = Sub(current.Apps, incoming.Apps)
	current.Containers = Sub(current.Containers, incoming.Containers)
	current.Jobs = Sub(current.Jobs, incoming.Jobs)
	current.Volumes = Sub(current.Volumes, incoming.Volumes)
	current.Images = Sub(current.Images, incoming.Images)

	current.Memory = SubQuantity(current.Memory, incoming.Memory)
	current.CPU = SubQuantity(current.CPU, incoming.CPU)

	// Only remove persistent resources if all is true.
	if all {
		current.VolumeStorage = SubQuantity(current.VolumeStorage, incoming.VolumeStorage)
	}
}

// Fits will check if a group of BaseResources will be able to contain
// another group of BaseResources. If the BaseResources are not able to fit,
// an aggregated error will be returned with all exceeded BaseResources.
// If the current BaseResources defines unlimited, then it will always fit.
func (current *BaseResources) Fits(incoming BaseResources) error {
	var exceededResources []string

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

	// Check if any of the quantity resources are exceeded
	for _, r := range []struct {
		resource          string
		current, incoming resource.Quantity
	}{
		{"VolumeStorage", current.VolumeStorage, incoming.VolumeStorage},
		{"Memory", current.Memory, incoming.Memory},
		{"Cpu", current.CPU, incoming.CPU},
	} {
		if !FitsQuantity(r.current, r.incoming) {
			exceededResources = append(exceededResources, r.resource)
		}
	}

	// Build an aggregated error message for the exceeded resources
	if len(exceededResources) > 0 {
		return fmt.Errorf("%w: %s", ErrExceededResources, strings.Join(exceededResources, ", "))
	}

	return nil
}

// ToString will return a string representation of the BaseResources within the struct.
func (current *BaseResources) ToString() string {
	return ResourcesToString(
		map[string]int{
			"Apps":       current.Apps,
			"Containers": current.Containers,
			"Jobs":       current.Jobs,
			"Volumes":    current.Volumes,
			"Images":     current.Images,
		},
		map[string]resource.Quantity{
			"VolumeStorage": current.VolumeStorage,
			"Memory":        current.Memory,
			"Cpu":           current.CPU,
		})
}

// Equals will check if the current BaseResources struct is equal to another. This is useful
// to avoid needing to do a deep equal on the entire struct.
func (current *BaseResources) Equals(incoming BaseResources) bool {
	return current.Apps == incoming.Apps &&
		current.Containers == incoming.Containers &&
		current.Jobs == incoming.Jobs &&
		current.Volumes == incoming.Volumes &&
		current.Images == incoming.Images &&
		current.VolumeStorage.Cmp(incoming.VolumeStorage) == 0 &&
		current.Memory.Cmp(incoming.Memory) == 0 &&
		current.CPU.Cmp(incoming.CPU) == 0
}
