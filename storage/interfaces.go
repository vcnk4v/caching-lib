// Package storage provides storage backends for the cache
package storage

import (
	"sync"
	"time"
)

// cache item with optional TTL
type Item[V any] struct {
	Value     V
	ExpiresAt time.Time
	HasTTL    bool
}

// checks if the item has expired
func (i *Item[V]) IsExpired() bool {
	return i.HasTTL && time.Now().After(i.ExpiresAt)
}

func (i *Item[V]) SetTTL(ttl time.Duration) {
	if ttl != 0 {
		i.ExpiresAt = time.Now().Add(ttl)
		i.HasTTL = true
	} else {
		i.HasTTL = false
	}
}

type ItemPool[V any] struct {
	pool sync.Pool
}

func NewItemPool[V any]() *ItemPool[V] {
	return &ItemPool[V]{
		pool: sync.Pool{
			New: func() interface{} {
				return &Item[V]{}
			},
		},
	}
}

// returns an item from the pool
func (p *ItemPool[V]) Get() *Item[V] {
	return p.pool.Get().(*Item[V])
}

// returns an item to the pool
func (p *ItemPool[V]) Put(item *Item[V]) {
	// Reset the item
	var zero V
	item.Value = zero
	item.HasTTL = false
	item.ExpiresAt = time.Time{}
	p.pool.Put(item)
}

// defines the interface for cache storage backends
type Storage[K comparable, V any] interface {
	Get(key K) (*Item[V], bool)
	Set(key K, item *Item[V])
	Delete(key K) bool
	Clear()
	Size() int
	Keys() []K
	// Cleanup expired items
	CleanupExpired() int
	// Reserve space for better memory efficiency
	Reserve(capacity int)
}
