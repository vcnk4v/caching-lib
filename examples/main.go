package main

import (
	"caching-lib/cache"
	"caching-lib/eviction"
	"fmt"
	"sync"
	"time"
)

// CustomRandomPolicy randomly evicts keys
type CustomRandomPolicy[K comparable] struct {
	items      map[K]struct{}
	keys       []K
	mu         sync.RWMutex
	threadSafe bool
}

func NewCustomRandomPolicy[K comparable](capacity int, threadSafe bool) *CustomRandomPolicy[K] {
	return &CustomRandomPolicy[K]{
		items:      make(map[K]struct{}, capacity),
		keys:       make([]K, 0, capacity),
		threadSafe: threadSafe,
	}
}

func (p *CustomRandomPolicy[K]) Access(key K) {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	if _, exists := p.items[key]; !exists {
		p.items[key] = struct{}{}
		p.keys = append(p.keys, key)
	}
}

func (p *CustomRandomPolicy[K]) Evict() (K, bool) {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	var zero K
	if len(p.keys) == 0 {
		return zero, false
	}

	// demo: pick middle
	index := len(p.keys) / 2
	key := p.keys[index]

	// Remove from both slice and map
	p.keys = append(p.keys[:index], p.keys[index+1:]...)
	delete(p.items, key)

	return key, true
}

func (p *CustomRandomPolicy[K]) Remove(key K) {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	if _, exists := p.items[key]; exists {
		delete(p.items, key)
		for i, k := range p.keys {
			if k == key {
				p.keys = append(p.keys[:i], p.keys[i+1:]...)
				break
			}
		}
	}
}

func (p *CustomRandomPolicy[K]) Clear() {
	if p.threadSafe {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	p.items = make(map[K]struct{})
	p.keys = p.keys[:0]
}

func (p *CustomRandomPolicy[K]) Size() int {
	if p.threadSafe {
		p.mu.RLock()
		defer p.mu.RUnlock()
	}

	return len(p.keys)
}

func main() {
	// Basic cache usage with LRU and generics
	fmt.Println("=== Basic Cache with LRU and Type Safety ===")
	c1 := cache.New(
		cache.WithCapacity[string, string](3),
		cache.WithEvictionPolicy[string, string](eviction.NewLRUWithConfig[string](3, true)),
		cache.WithThreadSafety[string, string](true),
	)

	c1.Set("key1", "value1")
	c1.Set("key2", "value2")
	c1.Set("key3", "value3")

	fmt.Printf("Cache size: %d\n", c1.Size())

	// Access key1 to make it recently used
	val, _ := c1.Get("key1")
	fmt.Printf("Retrieved: %s\n", val)

	// Add another key, should evict key2
	c1.Set("key4", "value4")

	if _, ok := c1.Get("key2"); !ok {
		fmt.Println("key2 was evicted (expected)")
	}

	// FIFO policy
	fmt.Println("\n=== Cache with FIFO ===")
	c2 := cache.New(
		cache.WithCapacity[string, string](2),
		cache.WithEvictionPolicy[string, string](eviction.NewFIFOWithConfig[string](2, true)),
	)

	c2.Set("first", "1")
	c2.Set("second", "2")
	c2.Set("third", "3") // Should evict "first"

	if _, ok := c2.Get("first"); !ok {
		fmt.Println("first was evicted (expected)")
	}

	// LIFO policy
	fmt.Println("\n=== Cache with LIFO ===")
	c3 := cache.New(
		cache.WithCapacity[string, string](2),
		cache.WithEvictionPolicy[string, string](eviction.NewLIFOWithConfig[string](2, true)),
	)

	c3.Set("first", "1")
	c3.Set("second", "2")
	c3.Set("third", "3") // Should evict "second"

	if _, ok := c3.Get("second"); !ok {
		fmt.Println("second was evicted (expected)")
	}

	// TTL support
	fmt.Println("\n=== Cache with TTL ===")
	c4 := cache.New(
		cache.WithCapacity[string, string](5),
		cache.WithDefaultTTL[string, string](2*time.Second),
		cache.WithThreadSafety[string, string](true),
	)

	c4.Set("temp1", "temporary value")
	c4.SetWithTTL("temp2", "expires quickly", 500*time.Millisecond)

	fmt.Printf("temp1 exists: %v\n", c4.Contains("temp1"))
	fmt.Printf("temp2 exists: %v\n", c4.Contains("temp2"))

	// Wait for temp2 to expire
	time.Sleep(600 * time.Millisecond)

	fmt.Printf("After 600ms - temp1 exists: %v\n", c4.Contains("temp1"))
	fmt.Printf("After 600ms - temp2 exists: %v\n", c4.Contains("temp2"))

	// Batch operations
	fmt.Println("\n===  Batch Operations ===")
	c5 := cache.New(cache.WithCapacity[string, string](10))

	// Set multiple items at once
	items := map[string]string{
		"batch1": "value1",
		"batch2": "value2",
		"batch3": "value3",
		"batch4": "value4",
	}

	setCount := c5.SetBatch(items)
	fmt.Printf("Set %d items in batch\n", setCount)

	// Get multiple items at once
	keys := []string{"batch1", "batch2", "batch3", "nonexistent"}
	results := c5.GetBatch(keys)

	fmt.Printf("Retrieved %d items: %v\n", len(results), results)

	// Delete multiple items at once
	deleteKeys := []string{"batch1", "batch2", "nonexistent"}
	deleted := c5.DeleteBatch(deleteKeys)
	fmt.Printf("Deleted %d items\n", deleted)

	// Cache statistics
	fmt.Println("\n=== Enhanced Cache Statistics ===")
	stats := c1.Stats()
	fmt.Printf("Hits: %d, Misses: %d, Evictions: %d\n",
		stats.Hits, stats.Misses, stats.Evictions)
	fmt.Printf("Hit Ratio: %.2f%%\n", stats.HitRatio*100)
	fmt.Printf("Size: %d, Capacity: %d\n", stats.Size, stats.Capacity)

	// Generic types
	fmt.Println("\n=== Generic Type Safety ===")

	// Integer cache
	intCache := cache.New(cache.WithCapacity[int, string](5))
	intCache.Set(1, "one")
	intCache.Set(2, "two")

	if val, ok := intCache.Get(1); ok {
		fmt.Printf("Integer key 1 -> %s\n", val)
	}

	// Struct cache
	type User struct {
		ID   int
		Name string
		Age  int
	}

	userCache := cache.New(cache.WithCapacity[string, User](5))
	user := User{ID: 1, Name: "John Doe", Age: 30}
	userCache.Set("user1", user)

	if retrievedUser, ok := userCache.Get("user1"); ok {
		fmt.Printf("User: %+v\n", retrievedUser)
	}

	// Thread-safe vs Non-thread-safe
	fmt.Println("\n=== Thread-Safe vs Non-Thread-Safe Performance ===")

	// Thread-safe cache
	threadSafeCache := cache.New(
		cache.WithCapacity[string, string](100),
		cache.WithThreadSafety[string, string](true),
	)

	// Non-thread-safe cache (for single-threaded use)
	nonThreadSafeCache := cache.New(
		cache.WithCapacity[string, string](100),
		cache.WithThreadSafety[string, string](false),
	)

	// Demonstrate concurrent access with thread-safe cache
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("concurrent_key_%d", i)
			threadSafeCache.Set(key, fmt.Sprintf("value_%d", i))
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("concurrent_key_%d", i)
			if val, ok := threadSafeCache.Get(key); ok {
				fmt.Printf("Concurrent read: %s -> %s\n", key, val)
			}
			time.Sleep(15 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	fmt.Printf("Final concurrent cache size: %d\n", threadSafeCache.Size())

	// Show non-thread-safe cache usage (single-threaded)
	nonThreadSafeCache.Set("single", "thread")
	fmt.Printf("Non-thread-safe cache size: %d\n", nonThreadSafeCache.Size())

	// Custom eviction policy
	fmt.Println("\n=== Custom Eviction Policy Example ===")

	// Using our custom random policy
	customCache := cache.New(
		cache.WithCapacity[string, string](3),
		cache.WithEvictionPolicy[string, string](NewCustomRandomPolicy[string](3, true)),
	)

	fmt.Println("Custom Random Policy:")
	customCache.Set("A", "1")
	customCache.Set("B", "2")
	customCache.Set("C", "3")

	// Add D, which should cause eviction
	customCache.Set("D", "4")

	// Check what was evicted
	remainingKeys := customCache.Keys()
	fmt.Printf("Remaining keys after custom eviction: %v\n", remainingKeys)

	// Compare eviction policies
	fmt.Println("\n===  Eviction Policy Comparison ===")
	testEvictionPolicy := func(name string, policy eviction.Policy[string]) {
		testCache := cache.New(
			cache.WithCapacity[string, string](3),
			cache.WithEvictionPolicy[string, string](policy),
		)

		fmt.Printf("\n%s Policy:\n", name)
		testCache.Set("A", "1")
		testCache.Set("B", "2")
		testCache.Set("C", "3")

		// Access A to affect LRU
		testCache.Get("A")

		// Add D, which should cause eviction
		testCache.Set("D", "4")

		// Check what was evicted
		remainingKeys := testCache.Keys()
		fmt.Printf("Remaining keys: %v\n", remainingKeys)
	}

	testEvictionPolicy(" LRU", eviction.NewLRUWithConfig[string](3, true))
	testEvictionPolicy(" FIFO", eviction.NewFIFOWithConfig[string](3, true))
	testEvictionPolicy(" LIFO", eviction.NewLIFOWithConfig[string](3, true))
	testEvictionPolicy("Custom Random", NewCustomRandomPolicy[string](3, true))

	// Memory efficiency
	fmt.Println("\n=== Memory Efficiency Demonstration ===")
	memoryEfficientCache := cache.New(
		cache.WithCapacity[string, string](1000),
		cache.WithThreadSafety[string, string](false),
	)

	// Fill cache to test memory pooling
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("mem_key_%d", i)
		memoryEfficientCache.Set(key, fmt.Sprintf("mem_value_%d", i))
	}

	// Clear and refill to test object pooling
	memoryEfficientCache.Clear()

	for i := 0; i < 300; i++ {
		key := fmt.Sprintf("reuse_key_%d", i)
		memoryEfficientCache.Set(key, fmt.Sprintf("reuse_value_%d", i))
	}

	fmt.Printf("Memory efficient cache final size: %d\n", memoryEfficientCache.Size())
}
