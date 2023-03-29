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
	ErrInvalidClass          = errors.New("compute class is invalid")
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

func ParseComputeClassMemory(memory ComputeClassMemory) (memoryQuantities, error) {
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
	for i, value := range memory.Values {
		valueInt, err := parseQuantity(value)
		if err != nil {
			return memoryQuantities{}, err
		}
		quantities.Values[i] = &valueInt
	}

	return quantities, nil
}

func CalculateCPU(wc ProjectComputeClassInstance, memDefault *int64, memory resource.Quantity) (resource.Quantity, error) {
	if err := ValidateComputeClass(wc, memory, memDefault); err != nil {
		return resource.Quantity{}, err
	}

	// The CPU scaler calculates the CPUs per Gi of memory so get the memory in a ratio of Gi
	memoryInGi := memory.AsApproximateFloat64() / gi
	// Since we're putting this in to mili-cpu's, multiply memoryInGi by the scaler and by 1000
	value := wc.CPUScaler * memoryInGi * 1000

	return *resource.NewMilliQuantity(int64(math.Ceil(value)), resource.DecimalSI), nil
}

func ValidateComputeClass(wc ProjectComputeClassInstance, memory resource.Quantity, memDefault *int64) error {
	parsedMemory, err := ParseComputeClassMemory(wc.Memory)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidClass, err)
	}

	if wc.Memory.Default != "" {
		wcDefault := parsedMemory.Def.Value()
		memDefault = &wcDefault
	}

	if !memoryInValues(parsedMemory, memory) {
		return fmt.Errorf("defined memory %v is not an allowed value for the ComputeClass %v. allowed values: %v",
			memory.String(), wc.Name, wc.Memory.Values)
	}

	memBytes := memory.Value()
	if max := parsedMemory.Max.Value(); max != 0 && memBytes > max {
		if memBytes == *memDefault {
			return fmt.Errorf("%w: default memory %v exceeds the maximum memory of %v for the ComputeClass %v",
				ErrInvalidMemoryForClass, memory.String(), parsedMemory.Max.String(), wc.Name)
		}
		return fmt.Errorf("%w: defined memory %v exceeds the maximum memory for the ComputeClass %v of %v",
			ErrInvalidMemoryForClass, memory.String(), wc.Name, parsedMemory.Max.String())
	}
	if min := parsedMemory.Min.Value(); memBytes != 0 && memBytes < min {
		if memBytes == *memDefault {
			return fmt.Errorf("%w: default memory %v is below the minimum memory of %v for the ComputeClass %v",
				ErrInvalidMemoryForClass, memory.String(), parsedMemory.Min.String(), wc.Name)
		}
		return fmt.Errorf("%w: defined memory %v is below the minimum memory for the ComputeClass %v of %v",
			ErrInvalidMemoryForClass, memory.String(), wc.Name, parsedMemory.Min.String())
	}

	return nil
}

func memoryInValues(parsedMemory memoryQuantities, memory resource.Quantity) bool {
	value := memory.Value()
	for _, allowedMemory := range parsedMemory.Values {
		if allowedMemory != nil && value == allowedMemory.Value() {
			return true
		}
	}
	return len(parsedMemory.Values) == 0
}

// GetClassForWorkload determines what ComputeClass should be used for the given appInstance, container and
// workload.
func GetClassForWorkload(ctx context.Context, c client.Client, computeClasses apiv1.ComputeClassMap, container apiv1.Container, workload, namespace string) (*ProjectComputeClassInstance, error) {
	wc, err := GetDefaultComputeClass(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	if specific, ok := computeClasses[workload]; ok {
		wc = specific
	} else if all, ok := computeClasses[""]; ok { // TODO - Will this always be the case?
		wc = all
	} else if container.ComputeClass != nil {
		wc = *container.ComputeClass
	}

	if wc == "" {
		return nil, nil
	}

	return GetAsProjectComputeClassInstance(ctx, c, namespace, wc)
}

// GetAsProjectComputeClassInstance grabs the ComputeClass spec (agnostic of it being cluster or project scoped) for the
// provided ComputeClass name.
func GetAsProjectComputeClassInstance(ctx context.Context, c client.Client, namespace string, computeClass string) (*ProjectComputeClassInstance, error) {
	if computeClass == "" {
		return nil, nil
	}

	projectComputeClass := ProjectComputeClassInstance{}
	err := c.Get(ctx, router.Key(namespace, computeClass), &projectComputeClass)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}

		clusterComputeClass := ClusterComputeClassInstance{}
		err = c.Get(ctx, router.Key("", computeClass), &clusterComputeClass)
		if err != nil {
			return nil, err
		}

		wc := ProjectComputeClassInstance(clusterComputeClass)
		return &wc, nil
	}
	wc := projectComputeClass
	return &wc, nil
}
