package system

import (
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Values will likely need to be tweaked as we get more usage data. They are currently based
// on metrics we have collected from internal use. You can override these values by setting
// the corresponding environment variable.
var (
	mi = int64(1_048_576) // 1 MiB in bytes

	registryMemoryRequest = *resource.NewQuantity(40*mi, resource.BinarySI)    // REGISTRY_MEMORY_REQUEST
	registryMemoryLimit   = *resource.NewQuantity(80*mi, resource.BinarySI)    // REGISTRY_MEMORY_LIMIT
	registryCPURequest    = *resource.NewMilliQuantity(50, resource.DecimalSI) // REGISTRY_CPU_REQUEST

	buildkitdMemoryRequest = *resource.NewQuantity(100*mi, resource.BinarySI)   // BUILDKITD_MEMORY_REQUEST
	buildkitdMemoryLimit   = *resource.NewQuantity(200*mi, resource.BinarySI)   // BUILDKITD_MEMORY_LIMIT
	buildkitdCPURequest    = *resource.NewMilliQuantity(50, resource.DecimalSI) // BUILDKITD_CPU_REQUEST

	buildkitdServiceMemoryRequest = *resource.NewQuantity(70*mi, resource.BinarySI)    // BUILDKITD_SERVICE_MEMORY_REQUEST
	buildkitdServiceMemoryLimit   = *resource.NewQuantity(140*mi, resource.BinarySI)   // BUILDKITD_SERVICE_MEMORY_LIMIT
	buildkitdServiceCPURequest    = *resource.NewMilliQuantity(50, resource.DecimalSI) // BUILDKITD_SERVICE_CPU_REQUEST
)

func RegistryResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: envOrDefault("REGISTRY_MEMORY_REQUEST", registryMemoryRequest),
			corev1.ResourceCPU:    envOrDefault("REGISTRY_CPU_REQUEST", registryCPURequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: envOrDefault("REGISTRY_MEMORY_LIMIT", registryMemoryLimit),
		},
	}
}

func BuildkitdResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: envOrDefault("BUILDKITD_MEMORY_REQUEST", buildkitdMemoryRequest),
			corev1.ResourceCPU:    envOrDefault("BUILDKITD_CPU_REQUEST", buildkitdCPURequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: envOrDefault("BUILDKITD_MEMORY_LIMIT", buildkitdMemoryLimit),
		},
	}
}

func BuildkitdServiceResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: envOrDefault("BUILDKITD_SERVICE_MEMORY_REQUEST", buildkitdServiceMemoryRequest),
			corev1.ResourceCPU:    envOrDefault("BUILDKITD_SERVICE_CPU_REQUEST", buildkitdServiceCPURequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: envOrDefault("BUILDKITD_SERVICE_MEMORY_LIMIT", buildkitdServiceMemoryLimit),
		},
	}
}

func envOrDefault(env string, def resource.Quantity) resource.Quantity {
	if env = os.Getenv(env); env == "" {
		return def
	}

	quantity, err := resource.ParseQuantity(env)
	if err == nil {
		return quantity
	}
	return def
}
