package appdefinition

import (
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/condition"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func OnError(req router.Request, _ router.Response, err error) error {
	if apierrors.IsConflict(err) {
		return err
	}
	// ignore and also don't record these errors
	if err != nil && strings.Contains(err.Error(), "object is being deleted") {
		return nil
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
