package stores

import (
	"fmt"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Builder struct {
	scheme *runtime.Scheme
	obj    kclient.Object

	List           strategy.Lister
	Update         strategy.Updater
	Get            strategy.Getter
	Create         strategy.Creater
	Delete         strategy.Deleter
	Destroy        strategy.Destroyer
	Watch          strategy.Watcher
	TableConverter rest.TableConvertor

	PrepareForUpdater strategy.PrepareForUpdater
	WarningsOnUpdater strategy.WarningsOnUpdater
	ValidateUpdater   strategy.ValidateUpdater

	PrepareForCreator strategy.PrepareForCreator
	WarningsOnCreator strategy.WarningsOnCreator
	Validator         strategy.Validator
	NameValidator     strategy.NameValidator
}

func NewBuilder(scheme *runtime.Scheme, obj kclient.Object) *Builder {
	return &Builder{
		scheme: scheme,
		obj:    obj,
	}
}

func (b Builder) WithWatch(watch strategy.Watcher) *Builder {
	b.Watch = watch
	return &b
}

func (b Builder) WithTableConverter(table rest.TableConvertor) *Builder {
	b.TableConverter = table
	return &b
}

func (b Builder) WithList(lister strategy.Lister) *Builder {
	b.List = lister
	return &b
}

func (b Builder) WithUpdate(update strategy.Updater) *Builder {
	b.Update = update
	return &b
}

func (b Builder) WithGet(get strategy.Getter) *Builder {
	b.Get = get
	return &b
}

func (b Builder) WithCompleteCRUD(complete strategy.CompleteCRUD) *Builder {
	return b.WithCreate(complete).
		WithGet(complete).
		WithList(complete).
		WithWatch(complete).
		WithUpdate(complete).
		WithDelete(complete)
}

func (b Builder) WithCreate(create strategy.Creater) *Builder {
	b.Create = create
	return &b
}

func (b Builder) WithPrepareUpdate(prepare strategy.PrepareForUpdater) *Builder {
	b.PrepareForUpdater = prepare
	return &b
}

func (b Builder) WithWarnOnUpdate(warn strategy.WarningsOnUpdater) *Builder {
	b.WarningsOnUpdater = warn
	return &b
}

func (b Builder) WithPrepareCreate(prepare strategy.PrepareForCreator) *Builder {
	b.PrepareForCreator = prepare
	return &b
}

func (b Builder) WithValidateUpdate(validate strategy.ValidateUpdater) *Builder {
	b.ValidateUpdater = validate
	return &b
}

func (b Builder) WithValidateCreate(validate strategy.Validator) *Builder {
	b.Validator = validate
	return &b
}

func (b Builder) WithValidateName(validate strategy.NameValidator) *Builder {
	b.NameValidator = validate
	return &b
}

func (b Builder) WithWarnOnCreate(warn strategy.WarningsOnCreator) *Builder {
	b.WarningsOnCreator = warn
	return &b
}

func (b Builder) WithDelete(deleter strategy.Deleter) *Builder {
	b.Delete = deleter
	return &b
}

func (b Builder) WithDestroy(destroy strategy.Destroyer) *Builder {
	b.Destroy = destroy
	return &b
}

func (b Builder) Build() rest.Storage {
	var (
		getSet    = b.Get != nil
		createSet = b.Create != nil
		updateSet = b.Update != nil
		listSet   = b.List != nil
		deleteSet = b.Delete != nil
		watchSet  = b.Watch != nil
	)

	if createSet && getSet && !listSet && !updateSet && !deleteSet && !watchSet {
		return &CreateGetStore{
			CreateAdapter:  b.createAdapter(),
			GetAdapter:     b.getAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if createSet && getSet && listSet && !updateSet && deleteSet && !watchSet {
		return &CreateGetListDeleteStore{
			GetAdapter:     b.getAdapter(),
			CreateAdapter:  b.createAdapter(),
			ListAdapter:    b.listAdapter(),
			DeleteAdapter:  b.deleteAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if createSet && !getSet && !listSet && !updateSet && !deleteSet && !watchSet {
		return &CreateOnlyStore{
			CreateAdapter:  b.createAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if !createSet && getSet && listSet && !updateSet && !deleteSet && !watchSet {
		return &GetListStore{
			GetAdapter:     b.getAdapter(),
			ListAdapter:    b.listAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			NewAdapter:     b.newAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if !createSet && getSet && listSet && !updateSet && deleteSet && !watchSet {
		return &GetListDeleteStore{
			GetAdapter:     b.getAdapter(),
			ListAdapter:    b.listAdapter(),
			DeleteAdapter:  b.deleteAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			NewAdapter:     b.newAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if !createSet && getSet && !listSet && !updateSet && !deleteSet && !watchSet {
		return &GetOnlyStore{
			GetAdapter:     b.getAdapter(),
			NewAdapter:     b.newAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			ScoperAdapter:  b.scoperAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if !createSet && !getSet && listSet && !updateSet && !deleteSet && !watchSet {
		return &ListOnlyStore{
			ListAdapter:    b.listAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			NewAdapter:     b.newAdapter(),
		}
	}
	if !createSet && getSet && listSet && !updateSet && deleteSet && watchSet {
		return &ReadDeleteStore{
			GetAdapter:     b.getAdapter(),
			ListAdapter:    b.listAdapter(),
			WatchAdapter:   b.watchAdapter(),
			DeleteAdapter:  b.deleteAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			NewAdapter:     b.newAdapter(),
		}
	}
	if createSet && getSet && listSet && updateSet && deleteSet && watchSet {
		return &ReadWriteWatchStore{
			CreateAdapter:  b.createAdapter(),
			GetAdapter:     b.getAdapter(),
			ListAdapter:    b.listAdapter(),
			UpdateAdapter:  b.updateAdapter(),
			DeleteAdapter:  b.deleteAdapter(),
			WatchAdapter:   b.watchAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if createSet && getSet && listSet && !updateSet && deleteSet && watchSet {
		return &CreateGetListDeleteWatchStore{
			CreateAdapter:  b.createAdapter(),
			GetAdapter:     b.getAdapter(),
			ListAdapter:    b.listAdapter(),
			DeleteAdapter:  b.deleteAdapter(),
			WatchAdapter:   b.watchAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if !createSet && getSet && listSet && !updateSet && deleteSet && !watchSet {
		return &GetListUpdateDeleteStore{
			GetAdapter:     b.getAdapter(),
			ListAdapter:    b.listAdapter(),
			UpdateAdapter:  b.updateAdapter(),
			DeleteAdapter:  b.deleteAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if !createSet && getSet && listSet && updateSet && deleteSet && watchSet {
		return &GetListUpdateDeleteWatchStore{
			GetAdapter:     b.getAdapter(),
			ListAdapter:    b.listAdapter(),
			UpdateAdapter:  b.updateAdapter(),
			DeleteAdapter:  b.deleteAdapter(),
			WatchAdapter:   b.watchAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	if !createSet && !getSet && listSet && !updateSet && !deleteSet && watchSet {
		return &ListWatchStore{
			NewAdapter:     b.newAdapter(),
			ListAdapter:    b.listAdapter(),
			WatchAdapter:   b.watchAdapter(),
			DestroyAdapter: b.destroyAdapter(),
			TableAdapter:   b.tableAdapter(),
		}
	}
	panic(fmt.Sprintf("createSet=%v, getSet=%v, listSet=%v, updateSet=%v, deleteSet=%v, watchSet=%v "+
		"combination is not currently supported, PRs welcomed!", createSet, getSet, listSet, updateSet, deleteSet,
		watchSet))
}

func (b Builder) watchAdapter() *strategy.WatchAdapter {
	return strategy.NewWatch(b.Watch)
}

type newer struct {
	obj kclient.Object
}

func (n *newer) New() types.Object {
	return n.obj.DeepCopyObject().(types.Object)
}

func (b Builder) newAdapter() *strategy.NewAdapter {
	return strategy.NewNew(&newer{obj: b.obj})
}

func (b Builder) scoperAdapter() *strategy.ScoperAdapter {
	return strategy.NewScoper(&newer{obj: b.obj})
}

func (b Builder) createAdapter() *strategy.CreateAdapter {
	create := strategy.NewCreate(b.scheme, b.Create)
	create.PrepareForCreater = b.PrepareForCreator
	create.Warner = b.WarningsOnCreator
	create.Validator = b.Validator
	create.NameValidator = b.NameValidator
	return create
}

func (b Builder) updateAdapter() *strategy.UpdateAdapter {
	update := strategy.NewUpdate(b.scheme, b.Update)
	update.PrepareForUpdater = b.PrepareForUpdater
	update.WarningsOnUpdater = b.WarningsOnUpdater
	update.ValidateUpdater = b.ValidateUpdater
	if b.Create != nil {
		update.CreateAdapter = b.createAdapter()
	}
	return update
}

func (b Builder) getAdapter() *strategy.GetAdapter {
	return strategy.NewGet(b.Get)
}

func (b Builder) destroyAdapter() *strategy.DestroyAdapter {
	return strategy.NewDestroyAdapter(b.Destroy)
}

func (b Builder) tableAdapter() *strategy.TableAdapter {
	return strategy.NewTable(b.TableConverter)
}

func (b Builder) listAdapter() *strategy.ListAdapter {
	return strategy.NewList(b.List)
}

func (b Builder) deleteAdapter() *strategy.DeleteAdapter {
	return strategy.NewDelete(b.scheme, b.Delete)
}
