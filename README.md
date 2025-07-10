# Caching Library

A high-performance, thread-safe in-memory cache for Go with pluggable eviction policies.

## Features

- **Type-safe** with Go generics
- **TTL support** with automatic expiration
- **Batch operations** for bulk set/get/delete
- **Thread-safe** with optional locking
- **Multiple eviction policies**: LRU, FIFO, LIFO
- **Statistics** with hit ratio tracking

## Quick Start

```go
import (
    "caching-lib/cache"
    "caching-lib/eviction"
)

// Create cache
c := cache.New[string, string](
    cache.WithCapacity[string, string](1000),
    cache.WithEvictionPolicy[string, string](eviction.NewLRU[string](1000)),
    cache.WithThreadSafety[string, string](true),
)

// Basic operations
c.Set("key", "value")
if value, exists := c.Get("key"); exists {
    fmt.Printf("Value: %s\n", value)
}
c.Delete("key")
```

## TTL Support

```go
// Set with TTL
c.SetWithTTL("temp", "data", 5*time.Minute)

// Default TTL for all items
c := cache.New[string, string](
    cache.WithCapacity[string, string](1000),
    cache.WithDefaultTTL[string, string](10*time.Minute),
)
```

## Batch Operations

```go
// Batch set
items := map[string]string{"key1": "value1", "key2": "value2"}
c.SetBatch(items)

// Batch get
results := c.GetBatch([]string{"key1", "key2"})

// Batch delete
c.DeleteBatch([]string{"key1", "key2"})
```

## Eviction Policies

```go
// LRU (default)
eviction.NewLRU[string](capacity)

// FIFO
eviction.NewFIFO[string](capacity)

// LIFO
eviction.NewLIFO[string](capacity)
```

### Custom Eviction Policy

Implement the `eviction.Policy[K]` interface:

```go
type CustomPolicy[K comparable] struct {
    // Your implementation
}

func (p *CustomPolicy[K]) Access(key K) {
    // Called when key is accessed
}

func (p *CustomPolicy[K]) Evict() (K, bool) {
    // Return key to evict and whether eviction occurred
}

func (p *CustomPolicy[K]) Remove(key K) {
    // Called when key is manually removed
}

func (p *CustomPolicy[K]) Clear() {
    // Clear all tracked keys
}

func (p *CustomPolicy[K]) Size() int {
    // Return number of tracked keys
}

// Use custom policy
c := cache.New[string, string](
    cache.WithCapacity[string, string](1000),
    cache.WithEvictionPolicy[string, string](&CustomPolicy[string]{}),
)
```

## Statistics

```go
stats := c.Stats()
fmt.Printf("Hit ratio: %.2f%%\n", stats.HitRatio*100)
fmt.Printf("Size: %d/%d\n", stats.Size, stats.Capacity)
```

## Configuration

```go
cache.New[K, V](
    cache.WithCapacity[K, V](size),
    cache.WithEvictionPolicy[K, V](policy),
    cache.WithThreadSafety[K, V](true),
    cache.WithDefaultTTL[K, V](duration),
)
```

## Structure

```
caching-lib/
├── cache/          # Core cache implementation
├── eviction/       # Eviction policy implementations
├── storage/        # Storage backend implementations
└── examples/       # Usage examples
```

## Testing

Run tests:

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./cache

# Run stress tests (takes longer)
go test -run=TestStress ./cache
```
