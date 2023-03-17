package pdb

import (
	appsv1 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ToPodDisruptionBudget(dep *appsv1.Deployment) *policyv1.PodDisruptionBudget {
	var maxUnavailable intstr.IntOrString
	if dep.Spec.Replicas == nil {
		maxUnavailable = intstr.FromString("25%")
	} else if *dep.Spec.Replicas > 2 {
		maxUnavailable = intstr.FromInt(int(*dep.Spec.Replicas-1) / 2)
	} else {
		// Replicas is 0 (app stopped) or 1
		maxUnavailable = intstr.FromInt(1)
	}

	return &policyv1.PodDisruptionBudget{
		ObjectMeta: dep.ObjectMeta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector:       dep.Spec.Selector,
			MaxUnavailable: &maxUnavailable,
		},
	}
}
