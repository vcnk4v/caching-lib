package eviction

import (
	"sync"
)

type Policy[K comparable] interface {
	// Access - called when key is accessed (get/set)
	Access(key K)

	// Evict - returns key to evict, or zero if nothing
	Evict() (K, bool)

	// Remove - called when key is manually deleted
	Remove(key K)

	// Clear - removes all tracked keys
	Clear()

	// Size - number of tracked keys
	Size() int
}

// shared item structure for all policies
type evictionItem[K comparable] struct {
	key K
}

// pool of reusable eviction items
type evictionItemPool[K comparable] struct {
	pool sync.Pool
}

// reates new eviction item pool
func newEvictionItemPool[K comparable]() *evictionItemPool[K] {
	return &evictionItemPool[K]{
		pool: sync.Pool{
			New: func() interface{} {
				return &evictionItem[K]{}
			},
		},
	}
}

// returns item from pool
func (p *evictionItemPool[K]) Get() *evictionItem[K] {
	return p.pool.Get().(*evictionItem[K])
}

// returns item to pool
func (p *evictionItemPool[K]) Put(item *evictionItem[K]) {
	var zero K
	item.key = zero
	p.pool.Put(item)
}
