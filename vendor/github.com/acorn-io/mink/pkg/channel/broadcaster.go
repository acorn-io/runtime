package channel

import (
	"context"
	"sync"
)

type Broadcaster[T any] struct {
	lock      sync.Mutex
	consumers map[*Subscription[T]]struct{}
	done      chan struct{}
	C         chan T
	closed    bool
}

func NewBroadcaster[T any](c chan T) *Broadcaster[T] {
	return &Broadcaster[T]{
		consumers: map[*Subscription[T]]struct{}{},
		done:      make(chan struct{}),
		C:         c,
	}
}

func (b *Broadcaster[T]) Start(ctx context.Context) {
	defer close(b.done)
	for {
		select {
		case <-ctx.Done():
			b.Close()
			return
		case i, ok := <-b.C:
			if !ok {
				b.lock.Lock()
				for sub := range b.consumers {
					sub.close(false)
				}
				b.lock.Unlock()
				return
			}
			b.lock.Lock()
			for sub := range b.consumers {
				sub.C <- i
			}
			b.lock.Unlock()
		}
	}
}

func (b *Broadcaster[T]) Shutdown() {
	b.lock.Lock()
	if b.closed {
		b.lock.Unlock()
		return
	}
	b.closed = true
	close(b.C)
	b.lock.Unlock()

	<-b.done
}

func (b *Broadcaster[T]) Close() {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.closed {
		return
	}
	for sub := range b.consumers {
		sub.close(false)
	}
	b.closed = true
	close(b.C)
}

func (b *Broadcaster[T]) Subscribe() *Subscription[T] {
	b.lock.Lock()
	defer b.lock.Unlock()
	c := make(chan T, 1)
	if b.closed {
		close(c)
		return &Subscription[T]{
			C:           c,
			broadcaster: b,
			closed:      true,
		}
	}
	sub := &Subscription[T]{
		C:           c,
		broadcaster: b,
	}
	b.consumers[sub] = struct{}{}
	return sub
}

type Subscription[T any] struct {
	C           chan T
	broadcaster *Broadcaster[T]
	closed      bool
}

func (s *Subscription[T]) Close() {
	s.close(true)
}

func (s *Subscription[T]) close(lock bool) {
	if lock {
		go func() {
			// empty the channel to ensure that the broadcaster is not blocking on writing to this subscription while
			// we wait for the broadcaster to release the lock
			for range s.C {
			}
		}()
		s.broadcaster.lock.Lock()
		defer s.broadcaster.lock.Unlock()
	}
	if s.closed {
		return
	}
	delete(s.broadcaster.consumers, s)
	close(s.C)
	s.closed = true
}
