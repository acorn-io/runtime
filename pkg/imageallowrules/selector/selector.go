package selector

import (
	"errors"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"
)

// UTIL

type LabelSelectorOpts struct {
	LabelRequirementErrorFilters []utilerrors.Matcher
}

func GenerateSelector(r v1.SignatureAnnotations, opts LabelSelectorOpts) (labels.Selector, error) {
	labelselector := &metav1.LabelSelector{
		MatchLabels:      r.Match,
		MatchExpressions: r.Expressions,
	}

	return labelSelectorAsSelector(labelselector, opts)
}

// labelSelectorAsSelector is adapted from k8s.io/apimachinery@v0.27.3/pkg/apis/meta/v1/helpers.go to include filtering of errors, e.g. to ignore the max length error for label values
func labelSelectorAsSelector(ps *metav1.LabelSelector, opts LabelSelectorOpts) (labels.Selector, error) {
	if ps == nil {
		return labels.Nothing(), nil
	}
	if len(ps.MatchLabels)+len(ps.MatchExpressions) == 0 {
		return labels.Everything(), nil
	}
	requirements := make([]labels.Requirement, 0, len(ps.MatchLabels)+len(ps.MatchExpressions))
	for k, v := range ps.MatchLabels {
		r, err := labels.NewRequirement(k, selection.Equals, []string{v})
		if utilerrors.FilterOut(err, opts.LabelRequirementErrorFilters...) != nil {
			return nil, err
		}
		requirements = append(requirements, *r)
	}
	for _, expr := range ps.MatchExpressions {
		var op selection.Operator
		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			op = selection.In
		case metav1.LabelSelectorOpNotIn:
			op = selection.NotIn
		case metav1.LabelSelectorOpExists:
			op = selection.Exists
		case metav1.LabelSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		default:
			return nil, fmt.Errorf("%q is not a valid label selector operator", expr.Operator)
		}
		r, err := labels.NewRequirement(expr.Key, op, append([]string(nil), expr.Values...))
		if utilerrors.FilterOut(err, opts.LabelRequirementErrorFilters...) != nil {
			return nil, err
		}
		requirements = append(requirements, *r)
	}
	selector := labels.NewSelector()
	selector = selector.Add(requirements...)
	return selector, nil
}

var LabelValueMaxLengthErrMsg string = validation.MaxLenError(validation.LabelValueMaxLength)

const LabelValueRegexpErrMsg string = "a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character"

func IgnoreInvalidFieldErrors(detailsPrefixes ...string) func(error) bool {
	return func(err error) bool {
		if ferr := (*field.Error)(nil); errors.As(err, &ferr) && ferr.Type == field.ErrorTypeInvalid {
			filteredErrs := slices.Filter(nil, strings.Split(ferr.Detail, ";"), func(s string) bool {
				for _, prefix := range detailsPrefixes {
					if strings.HasPrefix(strings.TrimSpace(s), prefix) {
						return false
					}
				}
				return true
			})
			return len(filteredErrs) == 0
		}
		return false
	}
}
