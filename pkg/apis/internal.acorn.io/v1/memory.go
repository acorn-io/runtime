package v1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acorn-io/aml/pkg/value"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ParseMemory(s []string) (MemoryMap, error) {
	result := MemoryMap{}
	for _, s := range s {
		workload, memBytes, specific := strings.Cut(s, "=")

		// If setting all, swap workload and memBytes
		if !specific {
			memBytes = workload
			workload = ""
		}

		quantity, err := value.Number(memBytes).ToInt()
		if err != nil {
			return MemoryMap{}, err
		}

		result[workload] = &quantity
	}
	return result, nil
}

var (
	ErrInvalidAcornMemory   = errors.New("invalid memory from Acornfile")
	ErrInvalidSetMemory     = errors.New("invalid memory set by user")
	ErrInvalidDefaultMemory = errors.New("invalid memory default")
	ErrInvalidWorkload      = errors.New("workload name set by user does not exist")
)

func ValidateMemory(memSpec MemoryMap, containerName string, container Container, specMemDefault, specMemMaximum *int64) (resource.Quantity, error) {
	var memMaximum, memDefault int64
	if specMemDefault != nil {
		memDefault = *specMemDefault
	}
	if specMemMaximum != nil {
		memMaximum = *specMemMaximum
	}

	// Determine which memory should be used to set the resource limit. Gets set
	// 4 ways: User setting a specific workload, user setting all workloads, Acornfile, or
	// from the apiv1.Config default.
	memBytes, errType := memDefault, ErrInvalidDefaultMemory
	if m, set := memSpec[containerName]; set && m != nil {
		errType = ErrInvalidSetMemory
		memBytes = *m
	} else if memSpec[""] != nil {
		errType = ErrInvalidSetMemory
		memBytes = *memSpec[""]
	} else if container.Memory != nil {
		errType = ErrInvalidAcornMemory
		memBytes = *container.Memory
	}

	// For maximum memory, 0 is equivalent to "unrestricted"
	var err error
	if memMaximum != 0 {
		var (
			maxQuantity     = resource.NewQuantity(memMaximum, resource.BinarySI).String()
			defaultQuantity = resource.NewQuantity(memDefault, resource.BinarySI).String()
			bytesQuantity   = resource.NewQuantity(memBytes, resource.BinarySI).String()
		)

		if memBytes > memMaximum {
			err = fmt.Errorf(
				"%w: workload \"%v\" with memory of %v exceeds the workload-memory-maximum of %v",
				errType, containerName, bytesQuantity, maxQuantity)
			if memBytes == memDefault {
				err = fmt.Errorf(
					"%w: workload-memory-default set to %v but exceeds the workload-memory-maximum of %v",
					errType, defaultQuantity, maxQuantity)
			}
		} else if memBytes == 0 {
			// For bytes, 0 is viewed as the maximum allowed memory. As such,
			// update to the current maximum.
			memBytes = memMaximum
		}
	}

	// Use the binary format for specifying memory (BinarySI)
	return *resource.NewQuantity(memBytes, resource.BinarySI), err
}
