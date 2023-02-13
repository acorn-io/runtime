package apply

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/acorn-io/baaah/pkg/apply/objectset"
	"github.com/acorn-io/baaah/pkg/merr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	LabelPrefix = "apply.acorn.io/"

	LabelSubContext = LabelPrefix + "owner-sub-context"
	LabelGVK        = LabelPrefix + "owner-gvk"
	LabelName       = LabelPrefix + "owner-name"
	LabelNamespace  = LabelPrefix + "owner-namespace"
	LabelHash       = LabelPrefix + "hash"

	AnnotationPrune  = LabelPrefix + "prune"
	AnnotationCreate = LabelPrefix + "create"
	AnnotationUpdate = LabelPrefix + "update"
)

var (
	hashOrder = []string{
		LabelSubContext,
		LabelGVK,
		LabelName,
		LabelNamespace,
	}
)

func (a *apply) apply(objs *objectset.ObjectSet) error {
	// retain the original order
	gvkOrder := objs.GVKOrder(a.knownGVK()...)

	labelSet, annotationSet, err := GetLabelsAndAnnotations(a.client.Scheme(), a.ownerSubContext, a.owner)
	if err != nil {
		return err
	}

	objs, err = a.injectLabelsAndAnnotations(objs, labelSet, annotationSet)
	if err != nil {
		return err
	}

	debugID := a.debugID()
	sel, err := GetSelector(labelSet)
	if err != nil {
		return err
	}

	var errs []error
	for _, gvk := range gvkOrder {
		err := a.process(debugID, sel, gvk, objs)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return merr.NewErrors(errs...)
}

func (a *apply) knownGVK() (ret []schema.GroupVersionKind) {
	for k := range a.pruneTypes {
		ret = append(ret, k)
	}
	return
}

func (a *apply) debugID() string {
	if a.owner == nil {
		return a.ownerSubContext
	}
	metadata, err := meta.Accessor(a.owner)
	if err != nil {
		return a.ownerSubContext
	}

	return fmt.Sprintf("%s %s", a.ownerSubContext, objectset.ObjectKey{
		Namespace: metadata.GetNamespace(),
		Name:      metadata.GetName(),
	})
}

func GetSelector(labelSet map[string]string) (labels.Selector, error) {
	if len(labelSet) == 0 {
		return nil, nil
	}
	req, err := labels.NewRequirement(LabelHash, selection.Equals, []string{labelSet[LabelHash]})
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*req), nil
}

func GetLabelsAndAnnotations(scheme *runtime.Scheme, ownerSubContext string, owner kclient.Object) (map[string]string, map[string]string, error) {
	if ownerSubContext == "" && owner == nil {
		return nil, nil, nil
	}

	annotations := map[string]string{
		LabelSubContext: ownerSubContext,
	}

	if ownerSubContext != "" {
		annotations[LabelSubContext] = ownerSubContext
	}

	if owner != nil {
		gvk, err := apiutil.GVKForObject(owner, scheme)
		if err != nil {
			return nil, nil, err
		}
		annotations[LabelGVK] = gvk.String()
		metadata, err := meta.Accessor(owner)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get metadata for %s", gvk)
		}
		annotations[LabelName] = metadata.GetName()
		annotations[LabelNamespace] = metadata.GetNamespace()
	}

	labels := map[string]string{
		LabelHash: objectSetHash(annotations),
	}

	return labels, annotations, nil
}

func (a *apply) injectLabelsAndAnnotations(in *objectset.ObjectSet, labels, annotations map[string]string) (*objectset.ObjectSet, error) {
	result, err := objectset.NewObjectSet(a.client.Scheme())
	if err != nil {
		return nil, err
	}

	for _, objMap := range in.ObjectsByGVK() {
		for _, obj := range objMap {
			if !a.ensure {
				obj = obj.DeepCopyObject().(kclient.Object)
			}
			setLabels(obj, labels)
			setAnnotations(obj, annotations)
			if err := result.Add(obj); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func setAnnotations(meta kclient.Object, annotations map[string]string) {
	objAnn := meta.GetAnnotations()
	if objAnn == nil {
		objAnn = map[string]string{}
	}
	delete(objAnn, LabelApplied)
	for k, v := range annotations {
		objAnn[k] = v
	}
	meta.SetAnnotations(objAnn)
}

func setLabels(meta kclient.Object, labels map[string]string) {
	objLabels := meta.GetLabels()
	if objLabels == nil {
		objLabels = map[string]string{}
	}
	for k, v := range labels {
		objLabels[k] = v
	}
	meta.SetLabels(objLabels)
}

func objectSetHash(labels map[string]string) string {
	dig := sha1.New()
	for _, key := range hashOrder {
		dig.Write([]byte(labels[key]))
	}
	return hex.EncodeToString(dig.Sum(nil))
}
