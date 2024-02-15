package volumes

import (
	"context"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/types"
	v1 "k8s.io/api/core/v1"
)

type PolicySwitcherStrategy struct {
	strategy   strategy.CompleteStrategy
	translator *Translator
	remote     *remote.Remote
}

// NewPolicySwitcherStrategy returns a new policy switcher strategy that switch persistvolume policy from retain to delete when the volume is being deleted.
// This makes sure the actual storage resource is deleted when the volume is deleted.
func NewPolicySwitcherStrategy(s strategy.CompleteStrategy, t *Translator, r *remote.Remote) strategy.Deleter {
	return &PolicySwitcherStrategy{
		translator: t,
		strategy:   s,
		remote:     r,
	}
}

func (p *PolicySwitcherStrategy) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	pvName, err := p.translator.FromVolumeToPVName(ctx, obj.GetNamespace(), obj.GetName())
	if err != nil {
		return nil, err
	}
	pvObj, err := p.remote.Get(ctx, "", pvName)
	if err != nil {
		return nil, err
	}
	pv := pvObj.(*v1.PersistentVolume)
	pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimDelete
	_, err = p.remote.Update(ctx, pv)
	if err != nil {
		return nil, err
	}

	return p.strategy.Delete(ctx, obj)
}

func (p *PolicySwitcherStrategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	return p.strategy.Get(ctx, namespace, name)
}
