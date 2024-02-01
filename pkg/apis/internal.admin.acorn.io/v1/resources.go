package v1

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/typed"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	ErrExceededResources        = fmt.Errorf("quota would be exceeded for resources")
	comparableUnlimitedQuantity = UnlimitedQuantity()
)

const Unlimited = -1

// NewUnlimitedQuantity creates a Quantity with an Unlimited value
func UnlimitedQuantity() resource.Quantity {
	return *resource.NewQuantity(Unlimited, resource.DecimalSI)
}

func Add(c, i int) int {
	if c == Unlimited || i == Unlimited {
		return Unlimited
	}
	return c + i
}

func AddQuantity(c, i resource.Quantity) resource.Quantity {
	if c.Equal(comparableUnlimitedQuantity) || i.Equal(comparableUnlimitedQuantity) {
		return UnlimitedQuantity()
	}
	c.Add(i)
	return c
}

func Sub(c, i int) int {
	if c == Unlimited || i == Unlimited {
		// We don't expect this situation to happen. This is because there should not be a situation
		// where we are removing from or with unlimited resources. However if it does, we want to
		// be careful and handle it. With that in mind the logic here is as follows:
		//
		// 1. If the current value is unlimited, then removing a non-unlimited value should not change
		//    the current value.
		// 2. If the current value is not unlimited, then removing an unlimited value should not
		//    change the current value.
		// 3. Finally if both values are unlimited, then the current value should remain unlimited.
		return c
	}

	// Ensure that we don't go below 0
	difference := c - i
	if difference < 0 {
		difference = 0
	}
	return difference
}

func SubQuantity(c, i resource.Quantity) resource.Quantity {
	// We don't expect this situation to happen by the same logic that is described in the Sub function.
	if c.Equal(comparableUnlimitedQuantity) || i.Equal(comparableUnlimitedQuantity) {
		return c
	}
	c.Sub(i)
	if c.CmpInt64(0) < 0 {
		return *resource.NewQuantity(0, c.Format)
	}
	return c
}

func Fits(current, incoming int) bool {
	if current != Unlimited && current < incoming {
		return false
	}
	return true
}

func FitsQuantity(current, incoming resource.Quantity) bool {
	if !current.Equal(comparableUnlimitedQuantity) && current.Cmp(incoming) < 0 {
		return false
	}
	return true
}

// CountResourcesToString will return a string representation of the resource and value
// if its value is greater than 0.
func CountResourcesToString(resources map[string]int) string {
	var resourceStrings []string

	for _, resource := range typed.Sorted(resources) {
		switch {
		case resource.Value > 0:
			resourceStrings = append(resourceStrings, fmt.Sprintf("%s: %d", resource.Key, resource.Value))
		case resource.Value == Unlimited:
			resourceStrings = append(resourceStrings, fmt.Sprintf("%s: unlimited", resource.Key))
		}
	}

	return strings.Join(resourceStrings, ", ")
}

func QuantityResourceToString(name string, quantity resource.Quantity) string {
	switch {
	case quantity.CmpInt64(0) > 0:
		return fmt.Sprintf("%s: %s", name, quantity.String())
	case quantity.Equal(comparableUnlimitedQuantity):
		return fmt.Sprintf("%s: unlimited", name)
	}
	return ""
}
