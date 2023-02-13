package router

import (
	"sync"

	"github.com/acorn-io/baaah/pkg/merr"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func isObjectForRequest(req Request, obj kclient.Object) bool {
	return obj.GetName() == req.Name &&
		obj.GetNamespace() == req.Namespace &&
		obj.GetObjectKind().GroupVersionKind() == req.GVK
}

type handlers struct {
	lock     sync.RWMutex
	handlers map[schema.GroupVersionKind][]Handler
}

func (h *handlers) GVKs() (result []schema.GroupVersionKind) {
	for gvk := range h.handlers {
		result = append(result, gvk)
	}
	return result
}

func (h *handlers) AddHandler(gvk schema.GroupVersionKind, handler Handler) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.handlers[gvk] = append(h.handlers[gvk], handler)
}

func (h *handlers) Handles(req Request) bool {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return len(h.handlers[req.GVK]) > 0
}

func (h *handlers) Handle(req Request, resp *response) error {
	h.lock.RLock()
	var (
		errs     []error
		handlers = h.handlers[req.GVK]
	)
	h.lock.RUnlock()

	for _, h := range handlers {
		err := h.Handle(req, resp)
		if err != nil {
			errs = append(errs, err)
		}
		newObjects := make([]kclient.Object, 0, len(resp.objects))
		for _, obj := range resp.objects {
			if isObjectForRequest(req, obj) {
				req.Object = obj
			} else {
				newObjects = append(newObjects, obj)
			}
		}
		resp.objects = newObjects
	}
	return merr.NewErrors(errs...)
}
