package v1

import (
	"context"
	"errors"
	"fmt"
	"math"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const gi = float64(1 << 30) // 1 Gibibyte. Note: this is not equivalent to 1 Gigabyte.

var (
	ErrInvalidMemoryForClass = errors.New("memory is invalid")
	ErrInvalidClass          = errors.New("workload class is invalid")
)

type memoryQuantities struct {
	Max    *resource.Quantity
	Min    *resource.Quantity
	Def    *resource.Quantity
	Values []*resource.Quantity
}

func parseQuantity(memory string) (resource.Quantity, error) {
	if memory == "" {
		memory = "0"
	}
	return resource.ParseQuantity(memory)
}

func ParseWorkloadClassMemory(memory WorkloadClassMemory) (memoryQuantities, error) {
	var quantities memoryQuantities

	minInt, err := parseQuantity(memory.Min)
	if err != nil {
		return memoryQuantities{}, err
	}
	quantities.Min = &minInt

	maxInt, err := parseQuantity(memory.Max)
	if err != nil {
		return memoryQuantities{}, err
	}
	quantities.Max = &maxInt

	defInt, err := parseQuantity(memory.Default)
	if err != nil {
		return memoryQuantities{}, err
	}
	quantities.Def = &defInt

	quantities.Values = make([]*resource.Quantity, len(memory.Values))
	for _, value := range memory.Values {
		valueInt, err := parseQuantity(value)
		if err != nil {
			return memoryQuantities{}, err
		}
		quantities.Values = append(quantities.Values, &valueInt)
	}

	return quantities, nil
}

func CalculateCPU(wc ProjectWorkloadClassInstance, memDefault *int64, memory resource.Quantity) (resource.Quantity, error) {
	if err := ValidateWorkloadClass(wc, memory, memDefault); err != nil {
		return resource.Quantity{}, err
	}

	// The CPU scaler calculates the CPUs per Gi of memory so get the memory in a ratio of Gi
	memoryInGi := memory.AsApproximateFloat64() / gi
	// Since we're putting this in to mili-cpu's, multiply memoryInGi by the scaler and by 1000
	value := wc.CPUScaler * memoryInGi * 1000

	return *resource.NewMilliQuantity(int64(math.Ceil(value)), resource.DecimalSI), nil
}

func ValidateWorkloadClass(wc ProjectWorkloadClassInstance, memory resource.Quantity, memDefault *int64) error {
	parsedMemory, err := ParseWorkloadClassMemory(wc.Memory)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidClass, err)
	}

	if wc.Memory.Default != "" {
		wcDefault := parsedMemory.Def.Value()
		memDefault = &wcDefault
	}

	if !memoryInValues(parsedMemory, memory) {
		return fmt.Errorf("defined memory %v is not an allowed value for the WorkloadClass %v. allowed values: %v",
			memory.String(), wc.Name, wc.Memory.Values)
	}

	memBytes := memory.Value()
	if max := parsedMemory.Max.Value(); max != 0 && memBytes > max {
		if memBytes == *memDefault {
			return fmt.Errorf("%w: default memory %v exceeds the maximum memory of %v for the WorkloadClass %v",
				ErrInvalidMemoryForClass, memory.String(), parsedMemory.Max.String(), wc.Name)
		}
		return fmt.Errorf("%w: defined memory %v exceeds the maximum memory for the WorkloadClass %v of %v",
			ErrInvalidMemoryForClass, memory.String(), wc.Name, parsedMemory.Max.String())

	}
	if min := parsedMemory.Min.Value(); memBytes != 0 && memBytes < min {
		if memBytes == *memDefault {
			return fmt.Errorf("%w: default memory %v is below the minimum memory of %v for the WorkloadClass %v",
				ErrInvalidMemoryForClass, memory.String(), parsedMemory.Min.String(), wc.Name)
		}
		return fmt.Errorf("%w: defined memory %v is below the minimum memory for the WorkloadClass %v of %v",
			ErrInvalidMemoryForClass, memory.String(), wc.Name, parsedMemory.Min.String())
	}

	return nil
}

func memoryInValues(parsedMemory memoryQuantities, memory resource.Quantity) bool {
	value := memory.Value()
	for _, allowedMemory := range parsedMemory.Values {
		if value == allowedMemory.Value() {
			return true
		}
	}
	return value == parsedMemory.Def.Value() ||
		value == parsedMemory.Max.Value() ||
		value == parsedMemory.Min.Value() ||
		len(parsedMemory.Values) == 0
}

// GetClassForWorkload determines what WorkloadClass should be used for the given appInstance, container and
// workload.
func GetClassForWorkload(ctx context.Context, c client.Client, workloadClasses apiv1.WorkloadClassMap, container apiv1.Container, workload, namespace string) (*ProjectWorkloadClassInstance, error) {
	wc, err := GetDefaultWorkloadClass(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	if specific, ok := workloadClasses[workload]; ok {
		wc = specific
	} else if all, ok := workloadClasses[""]; ok { // TODO - Will this always be the case?
		wc = all
	} else if container.WorkloadClass != nil {
		wc = *container.WorkloadClass
	}

	if wc == "" {
		return nil, nil
	}

	return GetAsProjectWorkloadClassInstance(ctx, c, namespace, wc)
}

// GetAsProjectWorkloadClassInstance grabs the WorkloadClass spec (agnostic of it being cluster or project scoped) for the
// provided WorkloadClass name.
func GetAsProjectWorkloadClassInstance(ctx context.Context, c client.Client, namespace string, workloadClass string) (*ProjectWorkloadClassInstance, error) {
	if workloadClass == "" {
		return nil, nil
	}

	projectWorkloadClass := ProjectWorkloadClassInstance{}
	err := c.Get(ctx, router.Key(namespace, workloadClass), &projectWorkloadClass)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}

		clusterWorkloadClass := ClusterWorkloadClassInstance{}
		err = c.Get(ctx, router.Key("", workloadClass), &clusterWorkloadClass)
		if err != nil {
			return nil, err
		}

		wc := ProjectWorkloadClassInstance(clusterWorkloadClass)
		return &wc, nil
	}
	wc := projectWorkloadClass
	return &wc, nil
}
