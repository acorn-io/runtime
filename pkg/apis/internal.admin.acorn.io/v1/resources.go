package v1

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

var ErrDoesNotFit = fmt.Errorf("quota would be exceeded for resources")

type (
	// Flags are boolean values that are either true or false, typically for features
	Flag string
	// Counts are integer values that are typically used for counting resources
	Count string
	// PersistentCounts are the sane as Counts, but are only subtracted from in Remove
	// if the all boolean is true.
	PersistentCount string
	// Quantities are resource.Quantity values that are typically used for measuring compute or
	// storage resources
	Quantity string

	Flags            map[Flag]bool
	Counts           map[Count]int
	PersistentCounts map[PersistentCount]int
	Quantities       map[Quantity]resource.Quantity
)

var (
	Unlimited Flag = "unlimited"

	Apps       Count = "apps"
	Containers Count = "containers"
	Jobs       Count = "jobs"
	Images     Count = "images"
	Projects   Count = "projects"

	Volumes PersistentCount = "volumes"
	Secrets PersistentCount = "secrets"

	VolumeStorage Quantity = "volumeStorage"
	Memory        Quantity = "memory"
	CPU           Quantity = "cpu"
)

// Resources is a struct separate from the QuotaRequestInstanceSpec to allow for
// external controllers to programmatically set the resources easier. Calls to
// its functions are mutating.
type Resources struct {
	Flags            Flags            `json:"flags,omitempty"`
	Counts           Counts           `json:"counts,omitempty"`
	PersistentCounts PersistentCounts `json:"persistentCounts,omitempty"`
	Quantities       Quantities       `json:"quantities,omitempty"`
}

// NewResources initializes and returns a new Resources instance with all maps initialized.
func NewResources() Resources {
	return Resources{
		Flags:            make(Flags),
		Counts:           make(Counts),
		PersistentCounts: make(PersistentCounts),
		Quantities:       make(Quantities),
	}
}

// Add will add the resources of another Resources struct into the current one.
func (current *Resources) Add(incoming Resources) {
	// Add the incoming resources to the current ones
	for flag, value := range incoming.Flags {
		current.Flags[flag] = value
	}
	for count, value := range incoming.Counts {
		current.Counts[count] += value
	}
	for quantity, value := range incoming.Quantities {
		q := current.Quantities[quantity]
		q.Add(value)
		current.Quantities[quantity] = q
	}
	for persistentCount, value := range incoming.PersistentCounts {
		current.PersistentCounts[persistentCount] += value
	}
}

// Remove will remove the resources of another Resources struct from the current one.
func (current *Resources) Remove(incoming Resources, all bool) {
	nonNegativeSubtract := func(a, b int) int {
		if a > b {
			return a - b
		}
		return 0
	}

	for c, value := range incoming.Counts {
		newValue := nonNegativeSubtract(current.Counts[c], value)
		if newValue == 0 {
			delete(current.Counts, c)
			continue
		}
		current.Counts[c] = newValue
	}

	for quantity, value := range incoming.Quantities {
		q := current.Quantities[quantity]
		q.Sub(value)
		current.Quantities[quantity] = q
	}

	// Don't proceed if persistent counts are not being removed
	if !all {
		return
	}

	for p, value := range incoming.PersistentCounts {
		newValue := nonNegativeSubtract(current.PersistentCounts[p], value)
		if newValue == 0 {
			delete(current.PersistentCounts, p)
			continue
		}
		current.PersistentCounts[p] = nonNegativeSubtract(current.PersistentCounts[p], value)
	}
}

// Fits will check if a group of resources will be able to contain
// another group of resources. If the resources are not able to fit,
// an aggregated error will be returned with all exceeded resources.
func (current *Resources) Fits(incoming Resources) error {
	exceededResources := []string{}

	if current.Flags[Unlimited] {
		return nil
	}

	for count, value := range incoming.Counts {
		if current.Counts[count] < value {
			exceededResources = append(exceededResources, string(count))
		}
	}
	for persistentCount, value := range incoming.PersistentCounts {
		if current.PersistentCounts[persistentCount] < value {
			exceededResources = append(exceededResources, string(persistentCount))
		}
	}
	for quantity, value := range incoming.Quantities {
		if q := current.Quantities[quantity]; q.Cmp(value) < 0 {
			exceededResources = append(exceededResources, string(quantity))
		}
	}

	// Build an aggregated error message for the exceeded resources
	if len(exceededResources) > 0 {
		return fmt.Errorf("%w: %s", ErrDoesNotFit, strings.Join(exceededResources, ", "))
	}

	return nil
}

func (current *Resources) IsEmpty() bool {
	return len(current.Flags) == 0 &&
		len(current.Counts) == 0 &&
		len(current.PersistentCounts) == 0 &&
		len(current.Quantities) == 0
}

// NonEmptyString will return a string representation of the non-empty
// Resources within the struct.
func (current *Resources) NonEmptyString() string {
	var resources []string

	for c, value := range current.Counts {
		if value > 0 {
			resources = append(resources, fmt.Sprintf("%v:%v", c, value))
		}
	}
	for p, value := range current.PersistentCounts {
		if value > 0 {
			resources = append(resources, fmt.Sprintf("%v:%v", p, value))
		}
	}
	for q, value := range current.Quantities {
		if !value.IsZero() {
			resources = append(resources, fmt.Sprintf("%v:%v", q, value))
		}
	}
	for p, value := range current.Flags {
		resources = append(resources, fmt.Sprintf("%v:%v", p, value))
	}

	return strings.Join(resources, ", ")
}

// Equals will check if the current Resources struct is equal to another. This is useful
// to avoid needing to do a deep equal on the entire struct.
func (current *Resources) Equals(incoming Resources) bool {
	// Convert both structs to JSON and compare the JSON representations.
	currentJSON, err1 := json.Marshal(current)
	incomingJSON, err2 := json.Marshal(incoming)

	if err1 != nil || err2 != nil {
		return false // Unable to marshal, return false.
	}

	return string(currentJSON) == string(incomingJSON)
}
