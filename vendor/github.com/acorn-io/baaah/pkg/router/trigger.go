package router

import (
	"strings"
	"sync"

	"github.com/acorn-io/baaah/pkg/backend"
	"github.com/acorn-io/baaah/pkg/uncached"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type triggers struct {
	lock      sync.RWMutex
	matchers  map[schema.GroupVersionKind]map[enqueueTarget][]matcher
	trigger   backend.Trigger
	gvkLookup backend.Backend
	scheme    *runtime.Scheme
	watcher   watcher
}

type watcher interface {
	WatchGVK(gvks ...schema.GroupVersionKind) error
}

type enqueueTarget struct {
	key string
	gvk schema.GroupVersionKind
}

func (m *triggers) invokeTriggers(req Request) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for enqueueTarget, matchers := range m.matchers[req.GVK] {
		if enqueueTarget.gvk == req.GVK &&
			enqueueTarget.key == req.Key {
			continue
		}
		for _, matcher := range matchers {
			if matcher.Match(req.GVK, req.Namespace, req.Name, req.Object) {
				logrus.Infof("Triggering [%s] [%v] from [%s] [%v]", enqueueTarget.key, enqueueTarget.gvk, req.Key, req.GVK)
				_ = m.trigger.Trigger(enqueueTarget.gvk, enqueueTarget.key, 0)
				break
			}
		}
	}
}

func (m *triggers) register(gvk schema.GroupVersionKind, key string, targetGVK schema.GroupVersionKind,
	mr matcher) {
	m.lock.Lock()
	defer m.lock.Unlock()

	target := enqueueTarget{
		key: key,
		gvk: gvk,
	}
	matchers, ok := m.matchers[targetGVK]
	if !ok {
		matchers = map[enqueueTarget][]matcher{}
		m.matchers[targetGVK] = matchers
	}
	for _, existing := range matchers[target] {
		if existing.Equals(mr) {
			return
		}
	}
	matchers[target] = append(matchers[target], mr)
}

func (m *triggers) Trigger(req Request, resp *response) error {
	if !req.FromTrigger {
		m.invokeTriggers(req)
	}
	return nil
}

func (m *triggers) Register(sourceGVK schema.GroupVersionKind, key string, obj runtime.Object, namespace, name string, selector labels.Selector, fields fields.Selector) (schema.GroupVersionKind, bool, error) {
	if uncached.IsWrapped(obj) {
		return schema.GroupVersionKind{}, false, nil
	}
	gvk, err := m.gvkLookup.GVKForObject(obj, m.scheme)
	if err != nil {
		return gvk, false, err
	}

	if _, ok := obj.(kclient.ObjectList); ok {
		gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	}

	m.register(sourceGVK, key, gvk, &objectMatcher{
		Namespace: namespace,
		Name:      name,
		Selector:  selector,
		Fields:    fields,
	})

	return gvk, true, m.watcher.WatchGVK(gvk)
}
