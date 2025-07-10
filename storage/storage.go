package storage

import (
	"sync"
)

// in-memory storage with map
type memoryStorage[K comparable, V any] struct {
	data       map[K]*Item[V]
	mu         sync.RWMutex
	itemPool   *ItemPool[V]
	threadSafe bool
}

// basic in-memory storage
func NewMemoryStorage[K comparable, V any]() Storage[K, V] {
	return &memoryStorage[K, V]{
		data:       make(map[K]*Item[V]),
		itemPool:   NewItemPool[V](),
		threadSafe: true, // thread-safe by default
	}
}

// storage with custom config
func NewMemoryStorageWithConfig[K comparable, V any](capacity int, threadSafe bool) Storage[K, V] {
	s := &memoryStorage[K, V]{
		data:       make(map[K]*Item[V], capacity),
		itemPool:   NewItemPool[V](),
		threadSafe: threadSafe,
	}
	return s
}

func (s *memoryStorage[K, V]) Get(key K) (*Item[V], bool) {
	if s.threadSafe {
		s.mu.RLock()
		defer s.mu.RUnlock()
	}

	item, exists := s.data[key]
	if !exists {
		return nil, false
	}

	if item.IsExpired() {
		// need write lock for delete
		if s.threadSafe {
			s.mu.RUnlock()
			s.mu.Lock()
			if item, exists := s.data[key]; exists && item.IsExpired() {
				delete(s.data, key)
				s.itemPool.Put(item)
			}
			s.mu.Unlock()
			s.mu.RLock()
		} else {
			delete(s.data, key)
			s.itemPool.Put(item)
		}
		return nil, false
	}

	return item, true
}

func (s *memoryStorage[K, V]) Set(key K, item *Item[V]) {
	if s.threadSafe {
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	// recycle old item
	if existing, exists := s.data[key]; exists {
		s.itemPool.Put(existing)
	}

	s.data[key] = item
}

func (s *memoryStorage[K, V]) Delete(key K) bool {
	if s.threadSafe {
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	if item, exists := s.data[key]; exists {
		delete(s.data, key)
		s.itemPool.Put(item)
		return true
	}
	return false
}

func (s *memoryStorage[K, V]) Clear() {
	if s.threadSafe {
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	// recycle everything
	for _, item := range s.data {
		s.itemPool.Put(item)
	}

	// clear without realloc
	for k := range s.data {
		delete(s.data, k)
	}
}

func (s *memoryStorage[K, V]) Size() int {
	if s.threadSafe {
		s.mu.RLock()
		defer s.mu.RUnlock()
	}

	return len(s.data)
}

func (s *memoryStorage[K, V]) Keys() []K {
	if s.threadSafe {
		s.mu.RLock()
		defer s.mu.RUnlock()
	}

	keys := make([]K, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// cleanup expired stuff
func (s *memoryStorage[K, V]) CleanupExpired() int {
	if s.threadSafe {
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	var removed int
	for key, item := range s.data {
		if item.IsExpired() {
			delete(s.data, key)
			s.itemPool.Put(item)
			removed++
		}
	}
	return removed
}

// pre-alloc for perf
func (s *memoryStorage[K, V]) Reserve(capacity int) {
	if s.threadSafe {
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	// make new map if empty
	if len(s.data) == 0 && capacity > 0 {
		s.data = make(map[K]*Item[V], capacity)
	}
}
