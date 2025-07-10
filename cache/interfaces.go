package cache

import (
	"caching-lib/eviction"
	"caching-lib/storage"
	"time"
)

// Cache - generic cache interface
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V) bool
	SetWithTTL(key K, value V, ttl time.Duration) bool
	Delete(key K) bool
	Clear()
	Size() int
	Keys() []K
	Stats() Stats
	Contains(key K) bool
	// batch ops
	SetBatch(items map[K]V) int
	GetBatch(keys []K) map[K]V
	DeleteBatch(keys []K) int
}

// Stats - cache metrics
type Stats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int
	Capacity  int
	HitRatio  float64
}

// Config - cache setup
type Config[K comparable, V any] struct {
	Capacity       int
	EvictionPolicy eviction.Policy[K]
	Storage        storage.Storage[K, V]
	ThreadSafe     bool
	DefaultTTL     time.Duration
	MaxTTL         time.Duration
}

type Option[K comparable, V any] func(*Config[K, V])

func WithCapacity[K comparable, V any](capacity int) Option[K, V] {
	return func(c *Config[K, V]) {
		if capacity <= 0 {
			capacity = 100 // default
		}
		c.Capacity = capacity
	}
}

func WithEvictionPolicy[K comparable, V any](policy eviction.Policy[K]) Option[K, V] {
	return func(c *Config[K, V]) {
		c.EvictionPolicy = policy
	}
}

func WithStorage[K comparable, V any](storage storage.Storage[K, V]) Option[K, V] {
	return func(c *Config[K, V]) {
		c.Storage = storage
	}
}

func WithThreadSafety[K comparable, V any](threadSafe bool) Option[K, V] {
	return func(c *Config[K, V]) {
		c.ThreadSafe = threadSafe
	}
}

func WithDefaultTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(c *Config[K, V]) {
		c.DefaultTTL = ttl
	}
}

func WithMaxTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(c *Config[K, V]) {
		c.MaxTTL = ttl
	}
}
