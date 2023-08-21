package v1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

var ErrExceededResources = fmt.Errorf("quota would be exceeded for resources")

const Unlimited = -1

// NewUnlimitedQuantity creates a Quantity with an Unlimited value
func UnlimitedQuantity() resource.Quantity {
	return *resource.NewQuantity(Unlimited, resource.DecimalSI)
}

// Resources is a struct separate from the QuotaRequestInstanceSpec to allow for
// external controllers to programmatically set the resources easier. Calls to
// its functions are mutating.
type Resources struct {
	Unlimited bool `json:"unlimited,omitempty"`

	Apps       int `json:"apps,omitempty"`
	Containers int `json:"containers,omitempty"`
	Jobs       int `json:"jobs,omitempty"`
	Volumes    int `json:"volumes,omitempty"`
	Secrets    int `json:"secrets,omitempty"`
	Images     int `json:"images,omitempty"`
	Projects   int `json:"projects,omitempty"`

	VolumeStorage resource.Quantity `json:"volumeStorage,omitempty"`
	Memory        resource.Quantity `json:"memory,omitempty"`
	CPU           resource.Quantity `json:"cpu,omitempty"`
}

// Add will add the resources of another Resources struct into the current one.
func (current *Resources) Add(incoming Resources) {
	add := func(c, i int) int {
		if c == Unlimited || i == Unlimited {
			return Unlimited
		}
		return c + i
	}

	unlimitedQuantity := UnlimitedQuantity()
	addQuantity := func(c, i resource.Quantity) resource.Quantity {
		if c.Equal(unlimitedQuantity) || i.Equal(unlimitedQuantity) {
			return unlimitedQuantity
		}
		c.Add(i)
		return c
	}

	current.Apps = add(current.Apps, incoming.Apps)
	current.Containers = add(current.Containers, incoming.Containers)
	current.Jobs = add(current.Jobs, incoming.Jobs)
	current.Volumes = add(current.Volumes, incoming.Volumes)
	current.Secrets = add(current.Secrets, incoming.Secrets)
	current.Images = add(current.Images, incoming.Images)
	current.Projects = add(current.Projects, incoming.Projects)

	current.VolumeStorage = addQuantity(current.VolumeStorage, incoming.VolumeStorage)
	current.Memory = addQuantity(current.Memory, incoming.Memory)
	current.CPU = addQuantity(current.CPU, incoming.CPU)
}

// Remove will remove the resources of another Resources struct from the current one. Calling remove
// will be a no-op for any resource values that are set to unlimited.
func (current *Resources) Remove(incoming Resources, all bool) {
	sub := func(c, i int) int {
		// We don't expect this situation to happen. This is because there should not be a situation
		// where we are removing from or with unlimited resources. However if it does, we want to
		// be careful and handle it. With that in mind the logic here is as follows:
		//
		// 1. If the current value is unlimited, then removing a non-unlimited value should not change
		//    the current value.
		// 2. If the current value is not unlimited, then removing an unlimited value should not
		//    change the current value.
		// 3. Finally if both values are unlimited, then the current value should remain unlimited.
		if c == Unlimited || i == Unlimited {
			return c
		}

		difference := c - i
		if difference < 0 {
			difference = 0
		}
		return difference
	}

	unlimitedQuantity := UnlimitedQuantity()
	subQuantity := func(c, i resource.Quantity) resource.Quantity {
		// This is the same situation as describe in the sub function above
		// but for the resource.Quantity type.
		if c.Equal(unlimitedQuantity) || i.Equal(unlimitedQuantity) {
			return c
		}

		c.Sub(i)
		if c.CmpInt64(0) < 0 {
			c.Set(0)
		}
		return c
	}

	current.Apps = sub(current.Apps, incoming.Apps)
	current.Containers = sub(current.Containers, incoming.Containers)
	current.Jobs = sub(current.Jobs, incoming.Jobs)
	current.Volumes = sub(current.Volumes, incoming.Volumes)
	current.Images = sub(current.Images, incoming.Images)
	current.Projects = sub(current.Projects, incoming.Projects)

	current.Memory = subQuantity(current.Memory, incoming.Memory)
	current.CPU = subQuantity(current.CPU, incoming.CPU)

	// Only remove persistent resources if all is true.
	if all {
		current.Secrets = sub(current.Secrets, incoming.Secrets)
		current.VolumeStorage = subQuantity(current.VolumeStorage, incoming.VolumeStorage)
	}
}

// Fits will check if a group of resources will be able to contain
// another group of resources. If the resources are not able to fit,
// an aggregated error will be returned with all exceeded resources.
// If the current resources defines unlimited, then it will always fit.
func (current *Resources) Fits(incoming Resources) error {
	if current.Unlimited {
		return nil
	}

	exceededResources := []string{}

	// Define function for checking int resources to keep code DRY
	checkResource := func(resource string, currentVal, incomingVal int) {
		if currentVal != Unlimited && currentVal < incomingVal {
			exceededResources = append(exceededResources, resource)
		}
	}

	// Define function for checking quantity resources to keep code DRY
	checkQuantityResource := func(resource string, currentVal, incomingVal resource.Quantity) {
		if !currentVal.Equal(UnlimitedQuantity()) && currentVal.Cmp(incomingVal) < 0 {
			exceededResources = append(exceededResources, resource)
		}
	}

	checkResource("Apps", current.Apps, incoming.Apps)
	checkResource("Containers", current.Containers, incoming.Containers)
	checkResource("Jobs", current.Jobs, incoming.Jobs)
	checkResource("Volumes", current.Volumes, incoming.Volumes)
	checkResource("Secrets", current.Secrets, incoming.Secrets)
	checkResource("Images", current.Images, incoming.Images)
	checkResource("Projects", current.Projects, incoming.Projects)

	checkQuantityResource("VolumeStorage", current.VolumeStorage, incoming.VolumeStorage)
	checkQuantityResource("Memory", current.Memory, incoming.Memory)
	checkQuantityResource("Cpu", current.CPU, incoming.CPU)

	// Build an aggregated error message for the exceeded resources
	if len(exceededResources) > 0 {
		return fmt.Errorf("%w: %s", ErrExceededResources, strings.Join(exceededResources, ", "))
	}

	return nil
}

// NonEmptyString will return a string representation of the non-empty
// Resources within the struct.
func (current *Resources) NonEmptyString() string {
	var resources []string

	// Define function for checking int resources to keep code DRY
	checkResource := func(resource string, value int) {
		if value > 0 {
			resources = append(resources, resource)
		}
	}

	// Define function for checking quantity resources to keep code DRY
	checkQuantityResource := func(resource string, currentVal resource.Quantity) {
		if !currentVal.IsZero() {
			resources = append(resources, resource)
		}
	}

	checkResource("Apps", current.Apps)
	checkResource("Containers", current.Containers)
	checkResource("Jobs", current.Jobs)
	checkResource("Volumes", current.Volumes)
	checkResource("Secrets", current.Secrets)
	checkResource("Images", current.Images)
	checkResource("Projects", current.Projects)

	checkQuantityResource("VolumeStorage", current.VolumeStorage)
	checkQuantityResource("Memory", current.Memory)
	checkQuantityResource("Cpu", current.CPU)

	return strings.Join(resources, ", ")
}

// Equals will check if the current Resources struct is equal to another. This is useful
// to avoid needing to do a deep equal on the entire struct.
func (current *Resources) Equals(incoming Resources) bool {
	return current.Unlimited == incoming.Unlimited &&
		current.Apps == incoming.Apps &&
		current.Containers == incoming.Containers &&
		current.Jobs == incoming.Jobs &&
		current.Volumes == incoming.Volumes &&
		current.Secrets == incoming.Secrets &&
		current.Images == incoming.Images &&
		current.Projects == incoming.Projects &&
		current.VolumeStorage.Cmp(incoming.VolumeStorage) == 0 &&
		current.Memory.Cmp(incoming.Memory) == 0 &&
		current.CPU.Cmp(incoming.CPU) == 0
}
