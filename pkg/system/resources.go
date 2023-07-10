package system

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"
)

var ErrInvalidResourceSpecification = fmt.Errorf("invalid resource specification")

// ValidateResources is used by the CLI to validate that the values passed
// can be parsed as a resource.Quantity and that the request is not higher
// than the limit.
func ValidateResources(resources ...string) error {
	for _, rs := range resources {
		if rs == "" {
			continue
		}

		req, limit, both := strings.Cut(rs, ":")
		if !both {
			_, err := resource.ParseQuantity(rs)
			return err
		}

		parsedReq, err := resource.ParseQuantity(req)
		if err != nil {
			return err
		}
		parsedLimit, err := resource.ParseQuantity(limit)
		if err != nil {
			return err
		}
		if parsedReq.Cmp(parsedLimit) > 0 {
			return fmt.Errorf("%w: resource request cannot be higher than the limit", ErrInvalidResourceSpecification)
		}
	}
	return nil
}

// ResourceRequirementsFor is used by components to create a ResourceRequirements struct
// based on strings found in the apiv1.Config struct.
func ResourceRequirementsFor(memory, cpu string) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{Requests: corev1.ResourceList{}, Limits: corev1.ResourceList{}}

	for resourceType, resourceString := range map[corev1.ResourceName]string{
		corev1.ResourceMemory: memory,
		corev1.ResourceCPU:    cpu,
	} {
		reqString, limitString, both := strings.Cut(resourceString, ":")

		if !both {
			parsedReq, err := resource.ParseQuantity(resourceString)
			if err == nil {
				requirements.Requests[resourceType] = parsedReq
			}
			continue
		}

		parsedReq, err := resource.ParseQuantity(reqString)
		if err == nil {
			requirements.Requests[resourceType] = parsedReq
		}

		parsedLimit, err := resource.ParseQuantity(limitString)
		if err == nil {
			requirements.Limits[resourceType] = parsedLimit
		}
	}

	return requirements
}
