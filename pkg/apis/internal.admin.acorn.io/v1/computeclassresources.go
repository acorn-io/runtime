package v1

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/typed"
	"k8s.io/apimachinery/pkg/api/resource"
)

// AllComputeClasses is a constant that can be used to define a ComputeResources struct that will apply to all
// ComputeClasses. This should only be used when defining a ComputeClassResources struct that is meant to be used
// as a limit and not a usage. The Fits method will work as expected when using this constant but Add and Remove
// do not interact with it.
const AllComputeClasses = "*"

type ComputeResources struct {
	Memory resource.Quantity `json:"memory,omitempty"`
	CPU    resource.Quantity `json:"cpu,omitempty"`
}

func (current *ComputeResources) Equals(incoming ComputeResources) bool {
	return current.Memory.Cmp(incoming.Memory) == 0 && current.CPU.Cmp(incoming.CPU) == 0
}

func (current *ComputeResources) ToString() string {
	var resourceStrings []string

	for _, r := range []struct {
		resource string
		value    resource.Quantity
	}{
		{"Memory", current.Memory},
		{"CPU", current.CPU},
	} {
		switch {
		case r.value.CmpInt64(0) > 0:
			resourceStrings = append(resourceStrings, fmt.Sprintf("%s: %s", r.resource, r.value.String()))
		case r.value.Equal(comparableUnlimitedQuantity):
			resourceStrings = append(resourceStrings, fmt.Sprintf("%s: unlimited", r.resource))
		}
	}

	return strings.Join(resourceStrings, ", ")
}

type ComputeClassResources map[string]ComputeResources

// Add will add the ComputeClassResources of another ComputeClassResources struct into the current one.
func (current ComputeClassResources) Add(incoming ComputeClassResources) {
	for computeClass, resources := range incoming {
		c := current[computeClass]
		c.Memory = AddQuantity(c.Memory, resources.Memory)
		c.CPU = AddQuantity(c.CPU, resources.CPU)
		current[computeClass] = c
	}
}

// Remove will remove the ComputeClassResources of another ComputeClassResources struct from the current one. Calling remove
// will be a no-op for any resource values that are set to unlimited.
func (current ComputeClassResources) Remove(incoming ComputeClassResources) {
	for computeClass, resources := range incoming {
		if _, ok := current[computeClass]; !ok {
			continue
		}

		c := current[computeClass]
		c.Memory = SubQuantity(c.Memory, resources.Memory)
		c.CPU = SubQuantity(c.CPU, resources.CPU)

		// Don't keep empty ComputeClasses
		if c.Equals(ComputeResources{}) {
			delete(current, computeClass)
		} else {
			current[computeClass] = c
		}
	}
}

// Fits will check if a group of ComputeClassResources will be able to contain
// another group of ComputeClassResources. If the ComputeClassResources are not able to fit,
// an aggregated error will be returned with all exceeded ComputeClassResources.
// If the current ComputeClassResources defines unlimited, then it will always fit.
func (current ComputeClassResources) Fits(incoming ComputeClassResources) error {
	var exceededResources []string

	// Check if any of the quantity resources are exceeded
	for computeClass, resources := range incoming {
		// If a specific compute class is defined on current then we check if it will
		// fit the incoming resources. If is not defined, then we check if the current
		// resources has AllComputeClasses defined and if so, we check if the incoming
		// resources will fit those. If neither are defined, then we deny the request
		// by appending the compute class to the exceeded resources and continuing.
		if _, ok := current[computeClass]; !ok {
			if _, ok := current[AllComputeClasses]; ok {
				computeClass = AllComputeClasses
			}
		}

		var ccExceededResources []string
		for _, r := range []struct {
			resource          string
			current, incoming resource.Quantity
		}{
			{"Memory", current[computeClass].Memory, resources.Memory},
			{"CPU", current[computeClass].CPU, resources.CPU},
		} {
			if !FitsQuantity(r.current, r.incoming) {
				ccExceededResources = append(ccExceededResources, r.resource)
			}
		}
		if len(ccExceededResources) > 0 {
			exceededResources = append(exceededResources, fmt.Sprintf("%q: %s", computeClass, strings.Join(ccExceededResources, ", ")))
		}
	}

	// Build an aggregated error message for the exceeded resources
	if len(exceededResources) > 0 {
		return fmt.Errorf("%w: ComputeClasses: %s", ErrExceededResources, strings.Join(exceededResources, ", "))
	}

	return nil
}

// ToString will return a string representation of the ComputeClassResources within the struct.
func (current ComputeClassResources) ToString() string {
	var resourceStrings []string

	for _, entry := range typed.Sorted(current) {
		resourceStrings = append(resourceStrings, fmt.Sprintf("%q: { %s }", entry.Key, entry.Value.ToString()))
	}

	return strings.Join(resourceStrings, ", ")
}

// Equals will check if the current ComputeClassResources struct is equal to another. This is useful
// to avoid needing to do a deep equal on the entire struct.
func (current ComputeClassResources) Equals(incoming ComputeClassResources) bool {
	if len(current) != len(incoming) {
		return false
	}

	for computeClass, resources := range incoming {
		if cc, ok := current[computeClass]; !ok || !cc.Equals(resources) {
			return false
		}
	}

	return true
}
