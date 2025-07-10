package eviction

import (
	"container/list"
	"sync"
)

// LIFO eviction policy
type lifoPolicy[K comparable] struct {
	capacity   int
	items      map[K]*list.Element
	order      *list.List
	mu         sync.RWMutex
	itemPool   *evictionItemPool[K]
	threadSafe bool
}

// NewLIFO - create LIFO policy
func NewLIFO[K comparable](capacity int) Policy[K] {
	return NewLIFOWithConfig[K](capacity, true)
}

// NewLIFOWithConfig - create LIFO with config
func NewLIFOWithConfig[K comparable](capacity int, threadSafe bool) Policy[K] {
	if capacity <= 0 {
		capacity = 100
	}
	return &lifoPolicy[K]{
		capacity:   capacity,
		items:      make(map[K]*list.Element, capacity),
		order:      list.New(),
		itemPool:   newEvictionItemPool[K](),
		threadSafe: threadSafe,
	}
}

func (p *lifoPolicy[K]) Access(key K) {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	if _, exists := p.items[key]; !exists {
		item := p.itemPool.Get()
		item.key = key
		elem := p.order.PushBack(item)
		p.items[key] = elem
	}
}

func (p *lifoPolicy[K]) Evict() (K, bool) {
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

func (p *lifoPolicy[K]) Remove(key K) {
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

func (p *lifoPolicy[K]) Clear() {
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

func (p *lifoPolicy[K]) Size() int {
	if p.threadSafe {
		p.mu.RLock()
		defer p.mu.RUnlock()
	}
	return len(p.items)
}
