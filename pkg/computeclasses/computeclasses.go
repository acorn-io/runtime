package computeclasses

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
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

func ParseComputeClassMemory(memory apiv1.ComputeClassMemory) (memoryQuantities, error) {
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

func ParseComputeClassMemoryInternal(memory internaladminv1.ComputeClassMemory) (memoryQuantities, error) {
	if memory.RequestScaler < 0 || memory.RequestScaler > 1 {
		return memoryQuantities{}, errors.New("request scaler value must be between 0 and 1, inclusive")
	}
	return ParseComputeClassMemory(apiv1.ComputeClassMemoryFromInternalAdmin(memory))
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

func Validate(cc apiv1.ComputeClass, memory resource.Quantity, memDefault *int64) error {
	parsedMemory, err := ParseComputeClassMemory(cc.Memory)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidClass, err)
	}

	if cc.Memory.Default != "" {
		wcDefault := parsedMemory.Def.Value()
		memDefault = &wcDefault
	}
	if !memoryInValues(parsedMemory, memory) {
		return fmt.Errorf("%w: defined memory %v is not an allowed value for the ComputeClass %v. allowed values: %v",
			ErrInvalidMemoryForClass, memory.String(), cc.Name, cc.Memory.Values)
	}

	memBytes := memory.Value()
	if max := parsedMemory.Max.Value(); max != 0 && memBytes > max {
		if memBytes == *memDefault {
			return fmt.Errorf("%w: default memory %v exceeds the maximum memory of %v for the ComputeClass %v",
				ErrInvalidMemoryForClass, memory.String(), parsedMemory.Max.String(), cc.Name)
		}
		return fmt.Errorf("%w: defined memory %v exceeds the maximum memory for the ComputeClass %v of %v",
			ErrInvalidMemoryForClass, memory.String(), cc.Name, parsedMemory.Max.String())
	}
	if min := parsedMemory.Min.Value(); memBytes != 0 && memBytes < min {
		if memBytes == *memDefault {
			return fmt.Errorf("%w: default memory %v is below the minimum memory of %v for the ComputeClass %v",
				ErrInvalidMemoryForClass, memory.String(), parsedMemory.Min.String(), cc.Name)
		}
		return fmt.Errorf("%w: defined memory %v is below the minimum memory for the ComputeClass %v of %v",
			ErrInvalidMemoryForClass, memory.String(), cc.Name, parsedMemory.Min.String())
	}

	return nil
}

func CalculateCPU(cc internaladminv1.ProjectComputeClassInstance, memory resource.Quantity) (resource.Quantity, error) {
	// The CPU scaler calculates the CPUs per Gi of memory so get the memory in a ratio of Gi
	memoryInGi := memory.AsApproximateFloat64() / gi
	// Since we're putting this in to mili-cpu's, multiply memoryInGi by the scaler and by 1000
	value := cc.CPUScaler * memoryInGi * 1000

	return *resource.NewMilliQuantity(int64(math.Ceil(value)), resource.DecimalSI), nil
}

func GetComputeClassNameForWorkload(workload string, container internalv1.Container, computeClasses internalv1.ComputeClassMap) string {
	var cc string
	if specific, ok := computeClasses[workload]; ok {
		cc = specific
	} else if all, ok := computeClasses[""]; ok { // TODO - Will this always be the case?
		cc = all
	} else if container.ComputeClass != nil {
		cc = *container.ComputeClass
	}

	return cc
}

// GetClassForWorkload determines what ComputeClass should be used for the given appInstance, container and
// workload.
func GetClassForWorkload(ctx context.Context, c client.Client, computeClasses internalv1.ComputeClassMap, container internalv1.Container, workload, namespace, region string) (*internaladminv1.ProjectComputeClassInstance, error) {
	var err error
	ccName := GetComputeClassNameForWorkload(workload, container, computeClasses)
	if ccName == "" {
		ccName, err = internaladminv1.GetDefaultComputeClass(ctx, c, namespace, region)
		if err != nil {
			return nil, err
		}
	}

	if ccName == "" {
		return nil, nil
	}

	return GetAsProjectComputeClassInstance(ctx, c, namespace, ccName)
}

// GetAsProjectComputeClassInstance grabs the ComputeClass spec (agnostic of it being cluster or project scoped) for the
// provided ComputeClass name.
func GetAsProjectComputeClassInstance(ctx context.Context, c client.Client, namespace string, computeClass string) (*internaladminv1.ProjectComputeClassInstance, error) {
	if computeClass == "" {
		return nil, nil
	}

	projectComputeClass := internaladminv1.ProjectComputeClassInstance{}
	err := c.Get(ctx, router.Key(namespace, computeClass), &projectComputeClass)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}

		clusterComputeClass := internaladminv1.ClusterComputeClassInstance{}
		err = c.Get(ctx, router.Key("", computeClass), &clusterComputeClass)
		if err != nil {
			return nil, err
		}

		cc := internaladminv1.ProjectComputeClassInstance(clusterComputeClass)
		return &cc, nil
	}
	return &projectComputeClass, nil
}
