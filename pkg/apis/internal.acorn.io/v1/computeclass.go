package v1

import (
	"strings"
)

func ParseComputeClass(s []string) (ComputeClassMap, error) {
	result := make(map[string]string, len(s))
	for _, s := range s {
		workload, computeClass, specific := strings.Cut(s, "=")

		// If setting all, swap workload and memBytes
		if !specific {
			computeClass = workload
			workload = ""
		}

		result[workload] = computeClass
	}
	return result, nil
}
