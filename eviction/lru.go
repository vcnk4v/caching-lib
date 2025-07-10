package eviction

import (
	"container/list"
	"sync"
)

// LRU eviction policy implementation
type lruPolicy[K comparable] struct {
	capacity   int
	items      map[K]*list.Element
	order      *list.List
	mu         sync.RWMutex
	itemPool   *evictionItemPool[K]
	threadSafe bool
}

// creates LRU policy
func NewLRU[K comparable](capacity int) Policy[K] {
	return NewLRUWithConfig[K](capacity, true)
}

// creates LRU with config
func NewLRUWithConfig[K comparable](capacity int, threadSafe bool) Policy[K] {
	if capacity <= 0 {
		capacity = 100
	}
	return &lruPolicy[K]{
		capacity:   capacity,
		items:      make(map[K]*list.Element, capacity),
		order:      list.New(),
		itemPool:   newEvictionItemPool[K](),
		threadSafe: threadSafe,
	}
}

// moves key to front (most recent)
func (p *lruPolicy[K]) Access(key K) {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	if elem, exists := p.items[key]; exists {
		p.order.MoveToFront(elem)
	} else {
		item := p.itemPool.Get()
		item.key = key
		elem := p.order.PushFront(item)
		p.items[key] = elem
	}
}

// removes and returns least recently used key
func (p *lruPolicy[K]) Evict() (K, bool) {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	var zero K
	if p.order.Len() == 0 {
		return zero, false
	}

	elem := p.order.Back()
	if elem != nil {
		p.order.Remove(elem)
		item := elem.Value.(*evictionItem[K])
		delete(p.items, item.key)
		key := item.key
		p.itemPool.Put(item)
		return key, true
	}

	return zero, false
}

// removes key from tracking
func (p *lruPolicy[K]) Remove(key K) {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	if elem, exists := p.items[key]; exists {
		p.order.Remove(elem)
		delete(p.items, key)
		item := elem.Value.(*evictionItem[K])
		p.itemPool.Put(item)
	}
}

func (p *lruPolicy[K]) Clear() {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	for elem := p.order.Front(); elem != nil; elem = elem.Next() {
		item := elem.Value.(*evictionItem[K])
		p.itemPool.Put(item)
	}

	for k := range p.items {
		delete(p.items, k)
	}
	p.order.Init()
}

// number of tracked keys
func (p *lruPolicy[K]) Size() int {
	if p.threadSafe {
		p.mu.RLock()
		defer p.mu.RUnlock()
	}

	return len(p.items)
}
