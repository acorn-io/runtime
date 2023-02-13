package apply

import (
	"errors"
	"fmt"
	"sort"

	"github.com/acorn-io/baaah/pkg/apply/objectset"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	ErrReplace = errors.New("replace object with changes")
)

func (a *apply) assignOwnerReference(gvk schema.GroupVersionKind, objs objectset.ObjectByKey) error {
	if a.owner == nil {
		return fmt.Errorf("no owner set to assign owner reference")
	}
	ownerMeta, err := meta.Accessor(a.owner)
	if err != nil {
		return err
	}
	ownerGVK, err := apiutil.GVKForObject(a.owner, a.client.Scheme())
	if err != nil {
		return err
	}
	ownerNSed, err := a.IsNamespaced(ownerGVK)
	if err != nil {
		return err
	}

	for k, v := range objs {
		// can't set owners across boundaries
		if ownerNSed {
			if nsed, err := a.IsNamespaced(gvk); err != nil {
				return err
			} else if !nsed {
				continue
			}
		}

		assignNS := false
		assignOwner := true
		if nsed, err := a.IsNamespaced(gvk); err != nil {
			return err
		} else if nsed {
			if k.Namespace == "" {
				assignNS = true
			} else if k.Namespace != ownerMeta.GetNamespace() && ownerNSed {
				assignOwner = false
			}
		}

		if !assignOwner {
			continue
		}

		v = v.DeepCopyObject().(kclient.Object)

		if assignNS {
			v.SetNamespace(ownerMeta.GetNamespace())
		}

		shouldSet := true
		for _, of := range v.GetOwnerReferences() {
			if ownerMeta.GetUID() == of.UID {
				shouldSet = false
				break
			}
		}

		if shouldSet && ownerMeta.GetUID() != "" {
			v.SetOwnerReferences(append(v.GetOwnerReferences(), metav1.OwnerReference{
				APIVersion:         ownerGVK.GroupVersion().String(),
				Kind:               ownerGVK.Kind,
				Name:               ownerMeta.GetName(),
				UID:                ownerMeta.GetUID(),
				Controller:         &[]bool{true}[0],
				BlockOwnerDeletion: &[]bool{true}[0],
			}))
		}

		objs[k] = v

		if assignNS {
			delete(objs, k)
			k.Namespace = ownerMeta.GetNamespace()
			objs[k] = v
		}
	}

	return nil
}

func (a *apply) adjustNamespace(objs objectset.ObjectByKey) error {
	for k, v := range objs {
		if k.Namespace != "" {
			continue
		}

		v.SetNamespace(a.defaultNamespace)

		delete(objs, k)
		k.Namespace = a.defaultNamespace
		objs[k] = v
	}

	return nil
}

func (a *apply) clearNamespace(objs objectset.ObjectByKey) error {
	for k, v := range objs {
		if k.Namespace == "" {
			continue
		}
		v.SetNamespace("")

		delete(objs, k)
		k.Namespace = ""
		objs[k] = v
	}

	return nil
}

func (a *apply) filterCrossVersion(objs *objectset.ObjectSet, gvk schema.GroupVersionKind, keys []objectset.ObjectKey) []objectset.ObjectKey {
	result := make([]objectset.ObjectKey, 0, len(keys))
	gk := gvk.GroupKind()
	for _, key := range keys {
		if objs.Contains(gk, key) {
			continue
		}
		if key.Namespace == a.defaultNamespace && objs.Contains(gk, objectset.ObjectKey{Name: key.Name}) {
			continue
		}
		result = append(result, key)
	}
	return result
}

func (a *apply) IsNamespaced(gvk schema.GroupVersionKind) (bool, error) {
	mapping, err := a.client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, err
	}
	return mapping.Scope.Name() == meta.RESTScopeNameNamespace, nil
}

func (a *apply) process(debugID string, set labels.Selector, gvk schema.GroupVersionKind, allObjs *objectset.ObjectSet) error {
	objs := allObjs.ObjectsByGVK()[gvk]

	nsed, err := a.IsNamespaced(gvk)
	if err != nil {
		return err
	}

	if a.owner != nil {
		if err := a.assignOwnerReference(gvk, objs); err != nil {
			return err
		}
	}

	if nsed {
		if err := a.adjustNamespace(objs); err != nil {
			return err
		}
	} else {
		if err := a.clearNamespace(objs); err != nil {
			return err
		}
	}

	existing, err := a.list(gvk, set, objs)
	if err != nil {
		return fmt.Errorf("failed to list %s for %s: %w", gvk, debugID, err)
	}

	toCreate, toDelete, toUpdate := compareSets(existing, objs)

	// check for resources in the objectset but under a different version of the same group/kind
	toDelete = a.filterCrossVersion(allObjs, gvk, toDelete)

	createF := func(k objectset.ObjectKey) error {
		obj, err := prepareObjectForCreate(gvk, objs[k], !a.ensure)
		if err != nil {
			return fmt.Errorf("failed to prepare create %s %s for %s: %w", k, gvk, debugID, err)
		}

		_, err = a.create(gvk, obj)
		if apierrors.IsAlreadyExists(err) {
			// Taking over an object that wasn't previously managed by us
			existingObj, getErr := a.get(gvk, objs[k], k.Namespace, k.Name)
			if getErr == nil {
				if existingObj.GetLabels()[LabelHash] != "" && !isAssigningSubContext(existingObj, obj) && !isAllowOwnerTransition(existingObj, obj) {
					return fmt.Errorf("failed to update existing owned object %s %s for %s, old subcontext [%s] gvk [%s] namespace [%s] name [%s]: %w", k, gvk, debugID,
						existingObj.GetAnnotations()[LabelSubContext],
						existingObj.GetAnnotations()[LabelGVK],
						existingObj.GetAnnotations()[LabelNamespace],
						existingObj.GetAnnotations()[LabelName], err)
				}
				if should(obj, AnnotationUpdate) {
					toUpdate = append(toUpdate, k)
				}
				existing[k] = existingObj
				return nil
			}
		}
		if err != nil {
			return fmt.Errorf("failed to create %s %s for %s: %w", k, gvk, debugID, err)
		}

		logrus.Debugf("DesiredSet - Created %s %s for %s", gvk, k, debugID)
		return nil
	}

	deleteF := func(k objectset.ObjectKey, force bool) error {
		if err := a.delete(gvk, k.Namespace, k.Name); err != nil {
			return fmt.Errorf("failed to delete %s %s for %s: %w", k, gvk, debugID, err)
		}
		logrus.Debugf("DesiredSet - DeleteStrategy %s %s for %s", gvk, k, debugID)
		return nil
	}

	updateF := func(k objectset.ObjectKey) error {
		err := a.compareObjects(gvk, debugID, existing[k], objs[k])
		if err == ErrReplace {
			if should(existing[k], AnnotationPrune) && should(existing[k], AnnotationCreate) {
				toDelete = append(toDelete, k)
				toCreate = append(toCreate, k)
			}
		} else if err != nil {
			return fmt.Errorf("failed to update %s %s for %s: %w", k, gvk, debugID, err)
		}
		return nil
	}

	var errs []error
	for _, k := range toCreate {
		errs = append(errs, createF(k))
	}
	toCreate = nil

	for _, k := range toUpdate {
		errs = append(errs, updateF(k))
	}

	if !a.noPrune {
		for _, k := range toDelete {
			errs = append(errs, deleteF(k, false))
		}
	}

	for _, k := range toCreate {
		errs = append(errs, createF(k))
	}

	return merr.NewErrors(errs...)
}

// isAllowedOwnerTransition is checking to see if an existing managed object
// was previously assigned with a subcontext that we want to allow to be changed
// to a different subcontext
func isAllowOwnerTransition(existingObj, newObj kclient.Object) bool {
	existingAnno := existingObj.GetAnnotations()
	newAnno := newObj.GetAnnotations()
	return newAnno[LabelSubContext] != "" &&
		existingAnno[LabelGVK] == newAnno[LabelGVK] &&
		existingAnno[LabelNamespace] == newAnno[LabelNamespace] &&
		existingAnno[LabelName] == newAnno[LabelName] &&
		validOwnerChange[fmt.Sprintf("%s => %s", existingAnno[LabelSubContext], newAnno[LabelSubContext])]

}

// isAssigningSubContext is checking to see if an existing managed object
// was previously assigned with no subcontext and is now trying to assign
// a subcontext.  We allow this as long as the previous owner is the same (gvk, namespace, name)
func isAssigningSubContext(existingObj, newObj kclient.Object) bool {
	existingAnno := existingObj.GetAnnotations()
	newAnno := newObj.GetAnnotations()
	return existingAnno[LabelSubContext] == "" &&
		newAnno[LabelSubContext] != "" &&
		existingAnno[LabelGVK] == newAnno[LabelGVK] &&
		existingAnno[LabelNamespace] == newAnno[LabelNamespace] &&
		existingAnno[LabelName] == newAnno[LabelName]
}

func (a *apply) list(gvk schema.GroupVersionKind, selector labels.Selector, objs map[objectset.ObjectKey]kclient.Object) (map[objectset.ObjectKey]kclient.Object, error) {
	if selector != nil {
		return a.listBySelector(gvk, selector)
	}

	result := map[objectset.ObjectKey]kclient.Object{}
	for k, v := range objs {
		obj, err := a.get(gvk, v, k.Namespace, k.Name)
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		result[k] = obj
	}

	return result, nil
}

func (a *apply) newObj(gvk schema.GroupVersionKind, list bool) (runtime.Object, error) {
	obj, err := a.client.Scheme().New(gvk)
	if runtime.IsNotRegisteredError(err) {
		if list {
			obj := &unstructured.UnstructuredList{}
			obj.SetGroupVersionKind(gvk)
			return obj, nil
		} else {
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(gvk)
			return obj, nil
		}
	} else if err != nil {
		return nil, err
	}
	return obj, nil
}

func (a *apply) listBySelector(gvk schema.GroupVersionKind, selector labels.Selector) (map[objectset.ObjectKey]kclient.Object, error) {
	var (
		errs []error
		objs = objectset.ObjectByKey{}
		list kclient.ObjectList
	)

	gvk.Kind += "List"
	obj, err := a.newObj(gvk, true)
	if err != nil {
		return nil, err
	}
	list = obj.(kclient.ObjectList)
	err = a.client.List(a.ctx, list, &kclient.ListOptions{
		Namespace:     a.listerNamespace,
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	err = meta.EachListItem(list, func(obj runtime.Object) error {
		if err := addObjectToMap(objs, obj.(kclient.Object)); err != nil {
			errs = append(errs, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return objs, merr.NewErrors(errs...)
}

func should(obj kclient.Object, label string) bool {
	return obj.GetAnnotations()[label] != "false"
}

func compareSets(existingSet, newSet objectset.ObjectByKey) (toCreate, toDelete, toUpdate []objectset.ObjectKey) {
	for k, obj := range newSet {
		if _, ok := existingSet[k]; ok {
			if should(obj, AnnotationCreate) {
				toUpdate = append(toUpdate, k)
			}
		} else {
			if should(obj, AnnotationUpdate) {
				toCreate = append(toCreate, k)
			}
		}
	}

	for k, obj := range existingSet {
		if _, ok := newSet[k]; !ok {
			if should(obj, AnnotationPrune) && obj.GetDeletionTimestamp().IsZero() {
				toDelete = append(toDelete, k)
			}
		}
	}

	sortObjectKeys(toCreate)
	sortObjectKeys(toDelete)
	sortObjectKeys(toUpdate)

	return
}

func sortObjectKeys(keys []objectset.ObjectKey) {
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
}

func addObjectToMap(objs objectset.ObjectByKey, obj kclient.Object) error {
	objs[objectset.ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}] = obj

	return nil
}
