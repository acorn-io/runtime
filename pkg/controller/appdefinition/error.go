package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/baaah/pkg/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func OnError(req router.Request, resp router.Response, err error) error {
	if apierrors.IsConflict(err) {
		return err
	}
	setCondition := false
	if _, ok := req.Object.(*v1.AppInstance); ok {
		setCondition = true
	}
	if _, ok := req.Object.(*v1.ServiceInstance); ok {
		setCondition = true
	}

	if !setCondition {
		return err
	}

	obj := req.Object
	oldObj := obj.DeepCopyObject().(kclient.Object)
	condition.ForName(obj, v1.AppInstanceConditionController).Error(err)

	updateErr := req.Get(oldObj, oldObj.GetNamespace(), oldObj.GetName())
	if updateErr == nil {
		if router.StatusChanged(oldObj, obj) {
			updateErr = req.Client.Status().Update(req.Ctx, obj)
		}
	}

	if updateErr == nil && apierrors.IsInvalid(err) {
		return nil
	}

	return err
}
