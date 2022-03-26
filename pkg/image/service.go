package image

import (
	"context"
	"sort"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"k8s.io/apimachinery/pkg/labels"
)

func Selector(l map[string]string) *ListOptions {
	return &ListOptions{
		Selector: labels.SelectorFromSet(l),
	}
}

type ListOptions struct {
	Selector labels.Selector
}

type registry struct {
	reg     name.Registry
	options []remote.Option
}

func (r *registry) List(ctx context.Context) ([]string, error) {
	repos, err := remote.Catalog(ctx, r.reg, r.options...)
	if err != nil {
		return nil, err
	}

	var (
		result     []string
		resultLock sync.Mutex
	)

	eg, ctx := errgroup.WithContext(ctx)
	sema := semaphore.NewWeighted(4)
	for _, repo := range repos {
		repo := repo
		rep, err := name.NewRepository(repo, name.WithDefaultRegistry(r.reg.RegistryStr()))
		if err != nil {
			return nil, err
		}
		if err := sema.Acquire(ctx, 1); err != nil {
			return nil, err
		}
		eg.Go(func() error {
			defer sema.Release(1)

			tags, err := remote.ListWithContext(ctx, rep, r.options...)
			if err != nil {
				return err
			}
			resultLock.Lock()
			for _, tag := range tags {
				ref, err := name.NewRepository(repo, name.WithDefaultRegistry(""))
				if err != nil {
					return err
				}
				result = append(result, ref.Tag(tag).String())
			}
			resultLock.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	sort.Strings(result)
	return result, nil
}
