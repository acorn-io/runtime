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
	Apps       int `json:"apps,omitempty"`
	Containers int `json:"containers,omitempty"`
	Jobs       int `json:"jobs,omitempty"`
	Volumes    int `json:"volumes,omitempty"`
	Secrets    int `json:"secrets,omitempty"`
	Images     int `json:"images,omitempty"`

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

	current.VolumeStorage.Add(incoming.VolumeStorage)
	current.Memory.Add(incoming.Memory)
	current.CPU.Add(incoming.CPU)
}

// Remove will remove the resources of another Resources struct from the current one.
func (current *Resources) Remove(incoming Resources, all bool) {
	current.Apps -= incoming.Apps
	current.Containers -= incoming.Containers
	current.Jobs -= incoming.Jobs
	current.Volumes -= incoming.Volumes
	current.Images -= incoming.Images

	current.Memory.Sub(incoming.Memory)
	current.CPU.Sub(incoming.CPU)

	// Only remove persistent resources if all is true.
	if all {
		current.Secrets -= incoming.Secrets
		current.VolumeStorage.Sub(incoming.VolumeStorage)
	}
}

// Fits will check if a group of resources will be able to contain
// another group of resources. If the resources are not able to fit,
// an aggregated error will be returned with all exceeded resources.
func (current *Resources) Fits(incoming Resources) error {
	exceededResources := []string{}
	if current.Apps < incoming.Apps {
		exceededResources = append(exceededResources, "Apps")
	}
	if current.Containers < incoming.Containers {
		exceededResources = append(exceededResources, "Containers")
	}
	if current.Jobs < incoming.Jobs {
		exceededResources = append(exceededResources, "Jobs")
	}
	if current.Volumes < incoming.Volumes {
		exceededResources = append(exceededResources, "Volumes")
	}
	if current.Secrets < incoming.Secrets {
		exceededResources = append(exceededResources, "Secrets")
	}
	if current.Images < incoming.Images {
		exceededResources = append(exceededResources, "Images")
	}

	if current.VolumeStorage.Cmp(incoming.VolumeStorage) < 0 {
		exceededResources = append(exceededResources, "VolumeStorage")
	}
	if current.Memory.Cmp(incoming.Memory) < 0 {
		exceededResources = append(exceededResources, "Memory")
	}
	if current.CPU.Cmp(incoming.CPU) < 0 {
		exceededResources = append(exceededResources, "Cpu")
	}

	// Build an aggregated error message for the exceeded resources
	if len(exceededResources) > 0 {
		return fmt.Errorf("exceeded quota for resources: %s", strings.Join(exceededResources, ", "))
	}

	return nil
}

// ToString will return a string representation of the Resources struct.
func (current *Resources) ToString() string {
	resources := []string{}
	if current.Apps > 0 {
		resources = append(resources, fmt.Sprintf("Apps:%v", current.Apps))
	}
	if current.Containers > 0 {
		resources = append(resources, fmt.Sprintf("Containers:%v", current.Containers))
	}
	if current.Jobs > 0 {
		resources = append(resources, fmt.Sprintf("Jobs:%v", current.Jobs))
	}
	if current.Volumes > 0 {
		resources = append(resources, fmt.Sprintf("Volumes:%v", current.Volumes))
	}
	if current.Secrets > 0 {
		resources = append(resources, fmt.Sprintf("Secrets:%v", current.Secrets))
	}
	if current.Images > 0 {
		resources = append(resources, fmt.Sprintf("Images:%v", current.Images))
	}

	if !current.VolumeStorage.IsZero() {
		resources = append(resources, fmt.Sprintf("VolumeStorage:%v", current.VolumeStorage))
	}
	if !current.Memory.IsZero() {
		resources = append(resources, fmt.Sprintf("Memory:%v", current.Memory))
	}
	if !current.CPU.IsZero() {
		resources = append(resources, fmt.Sprintf("CPU:%v", current.CPU))
	}

	return strings.Join(resources, ",")
}

// Equals will check if the current Resources struct is equal to another. This is useful
// to avoid needing to do a deep equal on the entire struct.
func (current *Resources) Equals(incoming Resources) bool {
	return current.Apps == incoming.Apps &&
		current.Containers == incoming.Containers &&
		current.Jobs == incoming.Jobs &&
		current.Volumes == incoming.Volumes &&
		current.Secrets == incoming.Secrets &&
		current.Images == incoming.Images &&
		current.VolumeStorage.Cmp(incoming.VolumeStorage) == 0 &&
		current.Memory.Cmp(incoming.Memory) == 0 &&
		current.CPU.Cmp(incoming.CPU) == 0
}
