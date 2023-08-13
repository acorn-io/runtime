package appstatus

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func isBlocked(dependencies map[string]v1.DependencyStatus, expressionErrors []v1.ExpressionError) (result []string, _ bool) {
	groupedByTypeName := map[string][]string{}

	for depName, dep := range dependencies {
		key := string(dep.DependencyType)
		if dep.Missing {
			key = string(dep.DependencyType) + " to be created"
		} else if !dep.Ready {
			key = string(dep.DependencyType) + " to be ready"
		}
		groupedByTypeName[key] = append(groupedByTypeName[key], depName)
	}

	for _, exprError := range expressionErrors {
		if exprError.DependencyNotFound != nil && exprError.DependencyNotFound.SubKey == "" {
			key := string(exprError.DependencyNotFound.DependencyType) + " to be created"
			groupedByTypeName[key] = append(groupedByTypeName[key], exprError.DependencyNotFound.Name)
		}
	}

	for _, key := range typed.SortedKeys(groupedByTypeName) {
		values := sets.NewString(groupedByTypeName[key]...).List()
		msg := fmt.Sprintf("waiting for %s [%s]", key, strings.Join(values, ", "))
		result = append(result, msg)
	}

	return result, len(result) > 0
}
