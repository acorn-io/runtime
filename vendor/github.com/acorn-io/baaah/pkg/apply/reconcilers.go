package apply

import (
	"encoding/json"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultReconcilers = map[schema.GroupVersionKind]reconciler{
		v1.SchemeGroupVersion.WithKind("Secret"):         reconcileSecret,
		v1.SchemeGroupVersion.WithKind("Service"):        reconcileService,
		batchv1.SchemeGroupVersion.WithKind("Job"):       reconcileJob,
		appsv1.SchemeGroupVersion.WithKind("Deployment"): reconcileDeployment,
		appsv1.SchemeGroupVersion.WithKind("DaemonSet"):  reconcileDaemonSet,
	}
)

func reconcileDaemonSet(oldObj, newObj kclient.Object) (bool, error) {
	oldSvc, ok := oldObj.(*appsv1.DaemonSet)
	if !ok {
		oldSvc = &appsv1.DaemonSet{}
		if err := convertObj(oldObj, oldSvc); err != nil {
			return false, err
		}
	}
	newSvc, ok := newObj.(*appsv1.DaemonSet)
	if !ok {
		newSvc = &appsv1.DaemonSet{}
		if err := convertObj(newObj, newSvc); err != nil {
			return false, err
		}
	}

	if !equality.Semantic.DeepEqual(oldSvc.Spec.Selector, newSvc.Spec.Selector) {
		return false, ErrReplace
	}

	return false, nil
}

func reconcileDeployment(oldObj, newObj kclient.Object) (bool, error) {
	oldSvc, ok := oldObj.(*appsv1.Deployment)
	if !ok {
		oldSvc = &appsv1.Deployment{}
		if err := convertObj(oldObj, oldSvc); err != nil {
			return false, err
		}
	}
	newSvc, ok := newObj.(*appsv1.Deployment)
	if !ok {
		newSvc = &appsv1.Deployment{}
		if err := convertObj(newObj, newSvc); err != nil {
			return false, err
		}
	}

	if !equality.Semantic.DeepEqual(oldSvc.Spec.Selector, newSvc.Spec.Selector) {
		return false, ErrReplace
	}

	return false, nil
}

func reconcileSecret(oldObj, newObj kclient.Object) (bool, error) {
	oldSvc, ok := oldObj.(*v1.Secret)
	if !ok {
		oldSvc = &v1.Secret{}
		if err := convertObj(oldObj, oldSvc); err != nil {
			return false, err
		}
	}
	newSvc, ok := newObj.(*v1.Secret)
	if !ok {
		newSvc = &v1.Secret{}
		if err := convertObj(newObj, newSvc); err != nil {
			return false, err
		}
	}

	if newSvc.Type != "" && oldSvc.Type != newSvc.Type {
		return false, ErrReplace
	}

	return false, nil
}

func reconcileService(oldObj, newObj kclient.Object) (bool, error) {
	oldSvc, ok := oldObj.(*v1.Service)
	if !ok {
		oldSvc = &v1.Service{}
		if err := convertObj(oldObj, oldSvc); err != nil {
			return false, err
		}
	}
	newSvc, ok := newObj.(*v1.Service)
	if !ok {
		newSvc = &v1.Service{}
		if err := convertObj(newObj, newSvc); err != nil {
			return false, err
		}
	}

	if newSvc.Spec.Type != "" && oldSvc.Spec.Type != newSvc.Spec.Type {
		return false, ErrReplace
	}

	return false, nil
}

func reconcileJob(oldObj, newObj kclient.Object) (bool, error) {
	oldJob, ok := oldObj.(*batchv1.Job)
	if !ok {
		oldJob = &batchv1.Job{}
		if err := convertObj(oldObj, oldJob); err != nil {
			return false, err
		}
	}

	newJob, ok := newObj.(*batchv1.Job)
	if !ok {
		newJob = &batchv1.Job{}
		if err := convertObj(newObj, newJob); err != nil {
			return false, err
		}
	}

	// We round trip the object here because when serializing to the applied
	// annotation values are truncated to 64 bytes.
	prunedSvc, err := getOriginalObject(newJob.GroupVersionKind(), newJob)
	if err != nil {
		return false, err
	}

	newPrunedJob := &batchv1.Job{}
	if err := convertObj(prunedSvc, newPrunedJob); err != nil {
		return false, err
	}

	if !equality.Semantic.DeepEqual(oldJob.Spec.Template, newPrunedJob.Spec.Template) {
		return false, ErrReplace
	}

	return false, nil
}

func convertObj(src interface{}, obj interface{}) error {
	uObj, ok := src.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("expected unstructured but got %v", reflect.TypeOf(src))
	}

	bytes, err := uObj.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, obj)
}
