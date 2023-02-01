package v1

import (
	"strings"
)

func ParseWorkloadClass(s []string) (WorkloadClassMap, error) {
	result := make(map[string]string, len(s))
	for _, s := range s {
		workload, workloadClass, specific := strings.Cut(s, "=")

		// If setting all, swap workload and memBytes
		if !specific {
			workloadClass = workload
			workload = ""
		}

		result[workload] = workloadClass
	}
	return result, nil
}
