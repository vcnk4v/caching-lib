package cache

import (
	"caching-lib/eviction"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	c := New(WithCapacity[string, string](3))

	c.Set("key1", "value1")
	if val, ok := c.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	if _, ok := c.Get("nonexistent"); ok {
		t.Error("Expected false for non-existent key")
	}

	if !c.Delete("key1") {
		t.Error("Expected true for successful delete")
	}

	if _, ok := c.Get("key1"); ok {
		t.Error("Expected false after delete")
	}
}

func TestCacheEviction(t *testing.T) {
	c := New(
		WithCapacity[string, string](2),
		WithEvictionPolicy[string, string](eviction.NewLRU[string](2)),
	)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3") // key1

	if _, ok := c.Get("key1"); ok {
		t.Error("Expected key1 to be evicted")
	}

	if _, ok := c.Get("key2"); !ok {
		t.Error("Expected key2 to still exist")
	}

	if _, ok := c.Get("key3"); !ok {
		t.Error("Expected key3 to exist")
	}
}

func TestCacheStats(t *testing.T) {
	c := New(WithCapacity[string, string](2))

	c.Set("key1", "value1")
	c.Get("key1") // hit
	c.Get("key2") // miss

	stats := c.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}

	if stats.HitRatio != 0.5 {
		t.Errorf("Expected hit ratio 0.5, got %f", stats.HitRatio)
	}
}

func TestCacheTTL(t *testing.T) {
	c := New(
		WithCapacity[string, string](5),
		WithDefaultTTL[string, string](100*time.Millisecond),
	)

	c.Set("key1", "value1")

	if val, ok := c.Get("key1"); !ok || val != "value1" {
		t.Error("Expected key1 to be available immediately")
	}

	c.SetWithTTL("key2", "value2", 50*time.Millisecond)

	time.Sleep(60 * time.Millisecond)

	if _, ok := c.Get("key2"); ok {
		t.Error("Expected key2 to be expired")
	}

	if _, ok := c.Get("key1"); !ok {
		t.Error("Expected key1 to still be available")
	}

	time.Sleep(60 * time.Millisecond)

	if _, ok := c.Get("key1"); ok {
		t.Error("Expected key1 to be expired")
	}
}

func TestCacheBatchOperations(t *testing.T) {
	c := New(WithCapacity[string, string](10))

	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	count := c.SetBatch(items)
	if count != 3 {
		t.Errorf("Expected 3 items set, got %d", count)
	}

	keys := []string{"key1", "key2", "key3", "nonexistent"}
	results := c.GetBatch(keys)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	if results["key1"] != "value1" {
		t.Errorf("Expected value1, got %s", results["key1"])
	}

	deleteKeys := []string{"key1", "key2", "nonexistent"}
	deleted := c.DeleteBatch(deleteKeys)
	if deleted != 2 {
		t.Errorf("Expected 2 items deleted, got %d", deleted)
	}

	if _, ok := c.Get("key1"); ok {
		t.Error("Expected key1 to be deleted")
	}

	if _, ok := c.Get("key3"); !ok {
		t.Error("Expected key3 to still exist")
	}
}

func TestCacheContains(t *testing.T) {
	c := New[string, string](WithCapacity[string, string](5))

	c.Set("key1", "value1")

	if !c.Contains("key1") {
		t.Error("Expected key1 to exist")
	}

	if c.Contains("nonexistent") {
		t.Error("Expected nonexistent key to not exist")
	}

	c.Delete("key1")

	if c.Contains("key1") {
		t.Error("Expected key1 to not exist after delete")
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := New(
		WithCapacity[string, string](100),
		WithThreadSafety[string, string](true),
	)

	var wg sync.WaitGroup
	numGoroutines := 50

	// concurrent writes
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				c.Set(key, fmt.Sprintf("value_%d_%d", id, j))
			}
		}(i)
	}

	// cncurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				c.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// for race
	if c.Size() > 100 {
		t.Errorf("Cache size exceeded capacity: %d", c.Size())
	}
}

func TestCacheGenericTypes(t *testing.T) {
	intCache := New(WithCapacity[int, int](5))
	intCache.Set(1, 100)
	intCache.Set(2, 200)

	if val, ok := intCache.Get(1); !ok || val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	type User struct {
		ID   int
		Name string
	}

	userCache := New(WithCapacity[string, User](5))
	user := User{ID: 1, Name: "John"}
	userCache.Set("user1", user)

	if retrievedUser, ok := userCache.Get("user1"); !ok || retrievedUser.Name != "John" {
		t.Errorf("Expected user John, got %v", retrievedUser)
	}
}

func TestCacheCapacityValidation(t *testing.T) {
	// Test with zero capacity - should default to 100
	c := New(WithCapacity[string, string](0))
	stats := c.Stats()
	if stats.Capacity != 100 {
		t.Errorf("Expected capacity 100, got %d", stats.Capacity)
	}

	// Test with negative capacity - should default to 100
	c2 := New(WithCapacity[string, string](-1))
	stats2 := c2.Stats()
	if stats2.Capacity != 100 {
		t.Errorf("Expected capacity 100, got %d", stats2.Capacity)
	}
}

func TestCacheEvictionPolicies(t *testing.T) {
	fifoCache := New(
		WithCapacity[string, string](2),
		WithEvictionPolicy[string, string](eviction.NewFIFO[string](2)),
	)

	fifoCache.Set("first", "1")
	fifoCache.Set("second", "2")
	fifoCache.Set("third", "3") // "first"

	if _, ok := fifoCache.Get("first"); ok {
		t.Error("Expected first to be evicted in FIFO")
	}

	// Test LIFO
	lifoCache := New(
		WithCapacity[string, string](2),
		WithEvictionPolicy[string, string](eviction.NewLIFO[string](2)),
	)

	lifoCache.Set("first", "1")
	lifoCache.Set("second", "2")
	lifoCache.Set("third", "3") // "second"

	if _, ok := lifoCache.Get("second"); ok {
		t.Error("Expected second to be evicted in LIFO")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	c := New(WithCapacity[string, string](1000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		c.Set(key, fmt.Sprintf("value_%d", i))
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := New(WithCapacity[string, string](1000))

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		c.Get(key)
	}
}

func BenchmarkCacheSetBatch(b *testing.B) {
	c := New(WithCapacity[string, string](1000))

	items := make(map[string]string)
	for i := 0; i < 100; i++ {
		items[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.SetBatch(items)
	}
}

func BenchmarkCacheGetBatch(b *testing.B) {
	c := New(WithCapacity[string, string](1000))

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, fmt.Sprintf("value_%d", i))
	}

	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("key_%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetBatch(keys)
	}
}

func BenchmarkCacheConcurrentAccess(b *testing.B) {
	c := New(
		WithCapacity[string, string](1000),
		WithThreadSafety[string, string](true),
	)

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("key_%d", b.N%1000)
			c.Get(key)
		}
	})
}
