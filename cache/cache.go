package cache

import (
	"caching-lib/eviction"
	"caching-lib/storage"
	"sync"
	"sync/atomic"
	"time"
)

type cache[K comparable, V any] struct {
	storage    storage.Storage[K, V]
	policy     eviction.Policy[K]
	capacity   int
	defaultTTL time.Duration
	maxTTL     time.Duration

	// stats (atomic for thread safety)
	hits      int64
	misses    int64
	evictions int64

	// thread safety
	mu         sync.RWMutex
	threadSafe bool

	// bg cleanup
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}

	// mem optimization
	itemPool *storage.ItemPool[V]
}

func New[K comparable, V any](opts ...Option[K, V]) Cache[K, V] {
	config := &Config[K, V]{
		Capacity:   100,
		ThreadSafe: true,
		DefaultTTL: 0, // no TTL by default
		MaxTTL:     24 * time.Hour,
	}

	for _, opt := range opts {
		opt(config)
	}

	if config.Storage == nil {
		config.Storage = storage.NewMemoryStorageWithConfig[K, V](config.Capacity, config.ThreadSafe)
	}

	if config.EvictionPolicy == nil {
		config.EvictionPolicy = eviction.NewLRUWithConfig[K](config.Capacity, config.ThreadSafe)
	}

	config.Storage.Reserve(config.Capacity)

	c := &cache[K, V]{
		storage:     config.Storage,
		policy:      config.EvictionPolicy,
		capacity:    config.Capacity,
		defaultTTL:  config.DefaultTTL,
		maxTTL:      config.MaxTTL,
		threadSafe:  config.ThreadSafe,
		stopCleanup: make(chan struct{}),
		itemPool:    storage.NewItemPool[V](),
	}

	// start bg cleanup if TTL enabled
	if config.DefaultTTL > 0 {
		c.startCleanup()
	}

	return c
}

// Get - retrieves value from cache
func (c *cache[K, V]) Get(key K) (V, bool) {
	if c.threadSafe {
		c.mu.RLock()
		defer c.mu.RUnlock()
	}

	var zero V
	item, exists := c.storage.Get(key)
	if exists && !item.IsExpired() {
		c.policy.Access(key)
		atomic.AddInt64(&c.hits, 1)
		return item.Value, true
	}

	atomic.AddInt64(&c.misses, 1)
	return zero, false
}

// Set - stores key-value pair
func (c *cache[K, V]) Set(key K, value V) bool {
	return c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL - stores with specific TTL
func (c *cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) bool {
	if c.threadSafe {
		c.mu.Lock()
		defer c.mu.Unlock()
	}

	if ttl > c.maxTTL {
		ttl = c.maxTTL
	}

	item := c.itemPool.Get()
	item.Value = value
	item.SetTTL(ttl)

	// check if key exists
	if existing, exists := c.storage.Get(key); exists && !existing.IsExpired() {
		c.storage.Set(key, item)
		c.policy.Access(key)
		return true
	}

	// evict if needed
	if c.storage.Size() >= c.capacity {
		if evictKey, hasKey := c.policy.Evict(); hasKey {
			c.storage.Delete(evictKey)
			atomic.AddInt64(&c.evictions, 1)
		}
	}

	// add new item
	c.storage.Set(key, item)
	c.policy.Access(key)

	return true
}

// Delete - removes key from cache
func (c *cache[K, V]) Delete(key K) bool {
	if c.threadSafe {
		c.mu.Lock()
		defer c.mu.Unlock()
	}

	if c.storage.Delete(key) {
		c.policy.Remove(key)
		return true
	}

	return false
}

// Clear - removes all items
func (c *cache[K, V]) Clear() {
	if c.threadSafe {
		c.mu.Lock()
		defer c.mu.Unlock()
	}

	c.storage.Clear()
	c.policy.Clear()
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)
	atomic.StoreInt64(&c.evictions, 0)
}

// current item count
func (c *cache[K, V]) Size() int {
	if c.threadSafe {
		c.mu.RLock()
		defer c.mu.RUnlock()
	}

	return c.storage.Size()
}

// all keys in cache
func (c *cache[K, V]) Keys() []K {
	if c.threadSafe {
		c.mu.RLock()
		defer c.mu.RUnlock()
	}

	return c.storage.Keys()
}

// checks if key exists
func (c *cache[K, V]) Contains(key K) bool {
	if c.threadSafe {
		c.mu.RLock()
		defer c.mu.RUnlock()
	}

	item, exists := c.storage.Get(key)
	return exists && !item.IsExpired()
}

// stores multiple items (memory optimized)
func (c *cache[K, V]) SetBatch(items map[K]V) int {
	if c.threadSafe {
		c.mu.Lock()
		defer c.mu.Unlock()
	}

	var count int

	// pre-allocate items from pool
	poolItems := make([]*storage.Item[V], 0, len(items))
	for range items {
		poolItems = append(poolItems, c.itemPool.Get())
	}

	i := 0
	for key, value := range items {
		if c.setBatchItem(key, value, poolItems[i]) {
			count++
		}
		i++
	}

	return count
}

// helper for batch ops (assumes lock held)
func (c *cache[K, V]) setBatchItem(key K, value V, item *storage.Item[V]) bool {
	item.Value = value
	item.SetTTL(c.defaultTTL)

	if existing, exists := c.storage.Get(key); exists && !existing.IsExpired() {
		c.storage.Set(key, item)
		c.policy.Access(key)
		return true
	}

	if c.storage.Size() >= c.capacity {
		if evictKey, hasKey := c.policy.Evict(); hasKey {
			c.storage.Delete(evictKey)
			atomic.AddInt64(&c.evictions, 1)
		}
	}

	c.storage.Set(key, item)
	c.policy.Access(key)
	return true
}

// retrieves multiple values
func (c *cache[K, V]) GetBatch(keys []K) map[K]V {
	if c.threadSafe {
		c.mu.RLock()
		defer c.mu.RUnlock()
	}

	result := make(map[K]V, len(keys))
	for _, key := range keys {
		if item, exists := c.storage.Get(key); exists && !item.IsExpired() {
			c.policy.Access(key)
			result[key] = item.Value
			atomic.AddInt64(&c.hits, 1)
		} else {
			atomic.AddInt64(&c.misses, 1)
		}
	}
	return result
}

// removes multiple keys
func (c *cache[K, V]) DeleteBatch(keys []K) int {
	if c.threadSafe {
		c.mu.Lock()
		defer c.mu.Unlock()
	}

	var count int
	for _, key := range keys {
		if c.storage.Delete(key) {
			c.policy.Remove(key)
			count++
		}
	}
	return count
}

// cache statistics
func (c *cache[K, V]) Stats() Stats {
	if c.threadSafe {
		c.mu.RLock()
		defer c.mu.RUnlock()
	}

	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)
	evictions := atomic.LoadInt64(&c.evictions)

	total := hits + misses
	var hitRatio float64
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}

	return Stats{
		Hits:      hits,
		Misses:    misses,
		Evictions: evictions,
		Size:      c.storage.Size(),
		Capacity:  c.capacity,
		HitRatio:  hitRatio,
	}
}

// starts bg cleanup of expired items
func (c *cache[K, V]) startCleanup() {
	cleanupInterval := c.defaultTTL / 2
	if cleanupInterval > time.Minute {
		cleanupInterval = time.Minute
	}
	if cleanupInterval < time.Second {
		cleanupInterval = time.Second
	}

	c.cleanupTicker = time.NewTicker(cleanupInterval)
	go func() {
		for {
			select {
			case <-c.cleanupTicker.C:
				c.cleanup()
			case <-c.stopCleanup:
				c.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// removes expired items
func (c *cache[K, V]) cleanup() {
	if c.threadSafe {
		c.mu.Lock()
		defer c.mu.Unlock()
	}

	c.storage.CleanupExpired()
}

// stops bg cleanup
func (c *cache[K, V]) Close() {
	if c.cleanupTicker != nil {
		close(c.stopCleanup)
	}
}
