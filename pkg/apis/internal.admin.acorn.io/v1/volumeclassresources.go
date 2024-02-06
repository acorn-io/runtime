package v1

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/typed"
	"k8s.io/apimachinery/pkg/api/resource"
)

// AllVolumeClasses is a constant that can be used to define a VolumeResources struct that will apply to all
// VolumeClasses. This should only be used when defining a VolumeClassResources struct that is meant to be used
// as a limit and not a usage. The Fits method will work as expected when using this constant but Add and Remove
// do not interact with it.
const AllVolumeClasses = "*"

type VolumeResources struct {
	VolumeStorage resource.Quantity `json:"volumeStorage"`
}

func (current *VolumeResources) ToString() string {
	switch {
	case current.VolumeStorage.CmpInt64(0) > 0:
		return "VolumeStorage: " + current.VolumeStorage.String()
	case current.VolumeStorage.Equal(comparableUnlimitedQuantity):
		return "VolumeStorage: unlimited"
	}
	return ""
}

type VolumeClassResources map[string]VolumeResources

// Add will add the VolumeClassResources of another VolumeClassResources struct into the current one.
func (current VolumeClassResources) Add(incoming VolumeClassResources) {
	for volumeClass, resources := range incoming {
		c := current[volumeClass]
		c.VolumeStorage = AddQuantity(c.VolumeStorage, resources.VolumeStorage)
		current[volumeClass] = c
	}
}

// Remove will remove the VolumeClassResources of another VolumeClassResources struct from the current one. Calling remove
// will be a no-op for any resource values that are set to unlimited.
func (current VolumeClassResources) Remove(incoming VolumeClassResources) {
	for volumeClass, resources := range incoming {
		if _, ok := current[volumeClass]; !ok {
			continue
		}

		c := current[volumeClass]
		c.VolumeStorage = SubQuantity(c.VolumeStorage, resources.VolumeStorage)

		// Don't keep empty VolumeClasses
		if c.VolumeStorage.CmpInt64(0) == 0 {
			delete(current, volumeClass)
		} else {
			current[volumeClass] = c
		}
	}
}

// Fits will check if a group of VolumeClassResources will be able to contain
// another group of VolumeClassResources. If the VolumeClassResources are not able to fit,
// an aggregated error will be returned with all exceeded VolumeClassResources.
// If the current VolumeClassResources defines unlimited, then it will always fit.
func (current VolumeClassResources) Fits(incoming VolumeClassResources) error {
	var exceededResources []string

	// Check if any of the quantity resources are exceeded
	for volumeClass, resources := range incoming {
		// If a specific volume class is defined on current then we check if it will
		// fit the incoming resources. If is not defined, then we check if the current
		// resources has AllVolumeClasses defined and if so, we check if the incoming
		// resources will fit those. If neither are defined, then we deny the request
		// by appending the volume class to the exceeded resources and continuing.
		if _, ok := current[volumeClass]; !ok {
			if _, ok := current[AllVolumeClasses]; ok {
				volumeClass = AllVolumeClasses
			}
		}

		if !FitsQuantity(current[volumeClass].VolumeStorage, resources.VolumeStorage) {
			exceededResources = append(exceededResources, fmt.Sprintf("%q: VolumeStorage", volumeClass))
		}
	}

	// Build an aggregated error message for the exceeded resources
	if len(exceededResources) > 0 {
		return fmt.Errorf("%w: VolumeClasses: %s", ErrExceededResources, strings.Join(exceededResources, ", "))
	}

	return nil
}

// ToString will return a string representation of the VolumeClassResources within the struct.
func (current VolumeClassResources) ToString() string {
	var resourceStrings []string

	for _, entry := range typed.Sorted(current) {
		resourceStrings = append(resourceStrings, fmt.Sprintf("%q: { %s }", entry.Key, entry.Value.ToString()))
	}

	return strings.Join(resourceStrings, ", ")
}

// Equals will check if the current VolumeClassResources struct is equal to another. This is useful
// to avoid needing to do a deep equal on the entire struct.
func (current VolumeClassResources) Equals(incoming VolumeClassResources) bool {
	if len(current) != len(incoming) {
		return false
	}

	for volumeClass, resources := range incoming {
		if c, ok := current[volumeClass]; !ok || !c.VolumeStorage.Equal(resources.VolumeStorage) {
			return false
		}
	}
	return true
}
