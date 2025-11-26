package optimistic

import (
	"sync"
	"time"
)

type ValueGetter[Key comparable, Value any] = func(Key) (Value, bool)

type OnDemandCache[Key comparable, Value any] struct {
	ttl     time.Duration
	locks   *sync.Map
	storage *sync.Map
	source  func(Key) (Value, bool)
	closer  chan struct{}
}

func NewCache[Key comparable, Value any](ttl time.Duration, getter ValueGetter[Key, Value]) *OnDemandCache[Key, Value] {
	self := &OnDemandCache[Key, Value]{
		locks:   new(sync.Map),
		storage: new(sync.Map),
		source:  getter,
		ttl:     ttl,
		closer:  make(chan struct{}),
	}

	return self
}

func (e *OnDemandCache[Key, Value]) Close() {
	close(e.closer)
}
func (c *OnDemandCache[K, V]) Get(key K) (V, bool) {
	v, ok := c.storage.Load(key)
	if ok {
		if v, isOk := v.(Entity[V]); isOk {
			return v.Value(), true
		}
	}

	tmp, isLoaded := c.locks.LoadOrStore(key, make(chan struct{}))

	barrier := tmp.(chan struct{})

	if isLoaded {
		<-barrier

		if v, ok := c.storage.Load(key); ok {
			if v, isOk := v.(Entity[V]); isOk {
				return v.Value(), true
			}
		}
		return *new(V), false
	}

	defer c.locks.Delete(key)
	defer close(barrier)

	nV, isOk := c.source(key)
	if !isOk {
		return *new(V), false
	}

	c.storage.Store(key, NewEntity[V](nV, c.ttl))

	return nV, true
}
