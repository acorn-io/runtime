package v1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

// Resources is a struct separate from the QuotaRequestInstanceSpec to allow for
// external controllers to programmatically set the resources easier. Calls to
// its functions are mutating.

type BaseResources struct {
	Apps       int `json:"apps,omitempty"`
	Containers int `json:"containers,omitempty"`
	Jobs       int `json:"jobs,omitempty"`
	Volumes    int `json:"volumes,omitempty"`
	Images     int `json:"images,omitempty"`

	VolumeStorage resource.Quantity `json:"volumeStorage,omitempty"`
	Memory        resource.Quantity `json:"memory,omitempty"`
	CPU           resource.Quantity `json:"cpu,omitempty"`
}

// Add will add the resources of another Resources struct into the current one.
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

// Remove will remove the resources of another Resources struct from the current one. Calling remove
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

// Fits will check if a group of resources will be able to contain
// another group of resources. If the resources are not able to fit,
// an aggregated error will be returned with all exceeded resources.
// If the current resources defines unlimited, then it will always fit.
func (current *BaseResources) Fits(incoming BaseResources) error {
	exceededResources := []string{}

	exceededResources = Fits(exceededResources, "Apps", current.Apps, incoming.Apps)
	exceededResources = Fits(exceededResources, "Containers", current.Containers, incoming.Containers)
	exceededResources = Fits(exceededResources, "Jobs", current.Jobs, incoming.Jobs)
	exceededResources = Fits(exceededResources, "Volumes", current.Volumes, incoming.Volumes)
	exceededResources = Fits(exceededResources, "Images", current.Images, incoming.Images)

	exceededResources = FitsQuantity(exceededResources, "VolumeStorage", current.VolumeStorage, incoming.VolumeStorage)
	exceededResources = FitsQuantity(exceededResources, "Memory", current.Memory, incoming.Memory)
	exceededResources = FitsQuantity(exceededResources, "Cpu", current.CPU, incoming.CPU)

	// Build an aggregated error message for the exceeded resources
	if len(exceededResources) > 0 {
		return fmt.Errorf("%w: %s", ErrExceededResources, strings.Join(exceededResources, ", "))
	}

	return nil
}

// ToString will return a string representation of the Resources within the struct.
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

// Equals will check if the current Resources struct is equal to another. This is useful
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
