package watcher

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type TransformFunc func(runtime.Object) []runtime.Object

func Transform(w watch.Interface, t TransformFunc) watch.Interface {
	result := &transformingWatch{
		transformFunc: t,
		realWatch:     w,
		realChan:      w.ResultChan(),
		resultChan:    make(chan watch.Event),
	}

	go func() {
		for event := range result.realChan {
			switch event.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				fallthrough
			case watch.Deleted:
				newObj := result.transformFunc(event.Object)
				for _, obj := range newObj {
					event.Object = obj
					result.resultChan <- event
				}
			default:
				result.resultChan <- event
			}
		}
		close(result.resultChan)
	}()

	return result
}

type transformingWatch struct {
	transformFunc TransformFunc
	realWatch     watch.Interface
	realChan      <-chan watch.Event
	resultChan    chan watch.Event
}

func (t *transformingWatch) Stop() {
	t.realWatch.Stop()
	go func() {
		for range t.realChan {
		}
	}()
}

func (t *transformingWatch) ResultChan() <-chan watch.Event {
	return t.resultChan
}
