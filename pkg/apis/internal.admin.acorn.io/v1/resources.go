package v1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

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
	current.Apps += incoming.Apps
	current.Containers += incoming.Containers
	current.Jobs += incoming.Jobs
	current.Volumes += incoming.Volumes
	current.Secrets += incoming.Secrets
	current.Images += incoming.Images
	current.Projects += incoming.Projects

	current.VolumeStorage.Add(incoming.VolumeStorage)
	current.Memory.Add(incoming.Memory)
	current.CPU.Add(incoming.CPU)
}

// Remove will remove the resources of another Resources struct from the current one.
func (current *Resources) Remove(incoming Resources, all bool) {
	// Do not allow resources to go below 0
	nonNegativeSubtract := func(currentVal, incomingVal int) int {
		difference := currentVal - incomingVal
		if difference < 0 {
			difference = 0
		}
		return difference
	}

	current.Apps = nonNegativeSubtract(current.Apps, incoming.Apps)
	current.Containers = nonNegativeSubtract(current.Containers, incoming.Containers)
	current.Jobs = nonNegativeSubtract(current.Jobs, incoming.Jobs)
	current.Volumes = nonNegativeSubtract(current.Volumes, incoming.Volumes)
	current.Images = nonNegativeSubtract(current.Images, incoming.Images)
	current.Projects = nonNegativeSubtract(current.Projects, incoming.Projects)

	current.Memory.Sub(incoming.Memory)
	current.CPU.Sub(incoming.CPU)

	// Only remove persistent resources if all is true.
	if all {
		current.Secrets = nonNegativeSubtract(current.Secrets, incoming.Secrets)
		current.VolumeStorage.Sub(incoming.VolumeStorage)
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
		if currentVal < incomingVal {
			exceededResources = append(exceededResources, resource)
		}
	}

	// Define function for checking quantity resources to keep code DRY
	checkQuantityResource := func(resource string, currentVal, incomingVal resource.Quantity) {
		if currentVal.Cmp(incomingVal) < 0 {
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
		return fmt.Errorf("quota would be exceeded for resources: %s", strings.Join(exceededResources, ", "))
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
