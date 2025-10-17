package optimistic

import (
	"fmt"
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

	go self.expireWatcher()
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

	mtx := sync.NewCond(new(sync.Mutex))

	_, isExists := c.locks.LoadOrStore(key, mtx)
	defer c.locks.Delete(key)

	if isExists {
		mtx.L.Lock()
		defer mtx.L.Unlock()
		mtx.Wait()

		if v, ok := c.storage.Load(key); ok {
			if v, isOk := v.(Entity[V]); isOk {
				return v.Value(), true
			}
		}
		return *new(V), false
	}

	nV, isOk := c.source(key)
	if !isOk {
		return *new(V), false
	}

	mtx.L.Lock()
	c.storage.Store(key, NewEntity[V](nV, c.ttl))
	mtx.L.Unlock()
	mtx.Broadcast()
	return nV, true
}

func (c *OnDemandCache[Key, Value]) expireWatcher() {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-c.closer:
			return
		case <-timer.C:
			ts := time.Now()
			c.storage.Range(func(k, v any) bool {
				ent, isOk := v.(Entity[Value])
				switch {
				case
					!isOk,
					!ent.BestBofore().IsZero() && ts.After(ent.BestBofore()):
					c.storage.Delete(k)
				}
				fmt.Println(ts.After(ent.BestBofore()))

				return true
			})
			timer.Reset(time.Second)
		}
	}
}
