package router

import (
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type FinalizerHandler struct {
	FinalizerID string
	Next        Handler
}

func (f FinalizerHandler) Handle(req Request, resp Response) error {
	obj := req.Object
	if obj == nil {
		return nil
	}

	if obj.GetDeletionTimestamp().IsZero() {
		if !slices.Contains(obj.GetFinalizers(), f.FinalizerID) {
			obj.SetFinalizers(append(obj.GetFinalizers(), f.FinalizerID))
			if err := req.Client.Update(req.Ctx, obj); err != nil {
				return err
			}
			resp.Objects(obj)
		}
		return nil
	}

	if len(obj.GetFinalizers()) == 0 || obj.GetFinalizers()[0] != f.FinalizerID {
		return nil
	}

	newResp := &ResponseWrapper{}
	newObj := obj.DeepCopyObject().(kclient.Object)
	req.Object = newObj

	if err := f.Next.Handle(req, newResp); err != nil {
		return err
	}

	if newResp.Delay != 0 {
		resp.RetryAfter(newResp.Delay)
	}

	for _, respObj := range newResp.Objs {
		if isObjectForRequest(req, respObj) {
			newObj = respObj
		}
		resp.Objects(respObj)
	}

	if StatusChanged(obj, newObj) {
		if err := req.Client.Status().Update(req.Ctx, newObj); err != nil {
			return err
		}
	}

	if newResp.Delay == 0 && len(newObj.GetFinalizers()) > 0 && newObj.GetFinalizers()[0] == f.FinalizerID {
		newObj.SetFinalizers(obj.GetFinalizers()[1:])
		if err := req.Client.Update(req.Ctx, newObj); err != nil {
			return err
		}
		resp.Objects(newObj)
	}

	return nil
}
