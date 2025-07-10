package cache

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"caching-lib/eviction"
)

// parallel Set 
func BenchmarkCacheSetParallel(b *testing.B) {
	c := New(
		WithCapacity[string, string](10000),
		WithThreadSafety[string, string](true),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i)
			c.Set(key, fmt.Sprintf("value_%d", i))
			i++
		}
	})
}

// parallel Get 
func BenchmarkCacheGetParallel(b *testing.B) {
	c := New(
		WithCapacity[string, string](10000),
		WithThreadSafety[string, string](true),
	)

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("key_%d", rand.Intn(10000))
			c.Get(key)
		}
	})
}

// mixed Get/Set operations
func BenchmarkCacheMixedParallel(b *testing.B) {
	c := New(
		WithCapacity[string, string](10000),
		WithThreadSafety[string, string](true),
	)

	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%3 == 0 {
				// 33% writes
				key := fmt.Sprintf("key_%d", i)
				c.Set(key, fmt.Sprintf("value_%d", i))
			} else {
				// 67% reads
				key := fmt.Sprintf("key_%d", rand.Intn(10000))
				c.Get(key)
			}
			i++
		}
	})
}

// compares eviction 
func BenchmarkEvictionPolicies(b *testing.B) {
	policies := map[string]eviction.Policy[string]{
		"LRU":  eviction.NewLRUWithConfig[string](1000, true),
		"FIFO": eviction.NewFIFOWithConfig[string](1000, true),
		"LIFO": eviction.NewLIFOWithConfig[string](1000, true),
	}

	for name, policy := range policies {
		b.Run(name, func(b *testing.B) {
			c := New(
				WithCapacity[string, string](1000),
				WithEvictionPolicy[string, string](policy),
			)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key_%d", i)
				c.Set(key, fmt.Sprintf("value_%d", i))
			}
		})
	}
}

// measures memory 
func BenchmarkMemoryUsage(b *testing.B) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	c := New(WithCapacity[string, string](100000))

	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, fmt.Sprintf("value_%d", i))
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/1024/1024, "MB")
}

// tests TTL overhead
func BenchmarkTTLOverhead(b *testing.B) {
	b.Run("WithoutTTL", func(b *testing.B) {
		c := New(WithCapacity[string, string](10000))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			c.Set(key, fmt.Sprintf("value_%d", i))
		}
	})

	b.Run("WithTTL", func(b *testing.B) {
		c := New(
			WithCapacity[string, string](10000),
			WithDefaultTTL[string, string](time.Hour),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			c.Set(key, fmt.Sprintf("value_%d", i))
		}
	})
}

// compares thread-safe vs non-thread-safe
func BenchmarkThreadSafetyOverhead(b *testing.B) {
	b.Run("ThreadSafe", func(b *testing.B) {
		c := New(
			WithCapacity[string, string](10000),
			WithThreadSafety[string, string](true),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			c.Set(key, fmt.Sprintf("value_%d", i))
		}
	})

	b.Run("NonThreadSafe", func(b *testing.B) {
		c := New(
			WithCapacity[string, string](10000),
			WithThreadSafety[string, string](false),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			c.Set(key, fmt.Sprintf("value_%d", i))
		}
	})
}

// batch operation performance
func BenchmarkBatchOperations(b *testing.B) {
	c := New(WithCapacity[string, string](10000))

	b.Run("SetBatch", func(b *testing.B) {
		items := make(map[string]string)
		for i := 0; i < 100; i++ {
			items[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.SetBatch(items)
		}
	})

	b.Run("GetBatch", func(b *testing.B) {
		// Pre-populate
		for i := 0; i < 1000; i++ {
			c.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
		}

		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			keys[i] = fmt.Sprintf("key_%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.GetBatch(keys)
		}
	})
}

// stress test
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	c := New(
		WithCapacity[string, string](10000),
		WithThreadSafety[string, string](true),
		WithDefaultTTL[string, string](time.Minute),
	)

	const (
		numGoroutines = 50
		numOperations = 1000
	)

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Start multiple goroutines performing different operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)

				switch j % 4 {
				case 0: // Set
					c.Set(key, value)
				case 1: // Get
					c.Get(key)
				case 2: // Delete
					c.Delete(key)
				case 3: // Contains
					c.Contains(key)
				}
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 100; i++ {
			items := make(map[string]string)
			for j := 0; j < 10; j++ {
				items[fmt.Sprintf("batch_%d_%d", i, j)] = fmt.Sprintf("bvalue_%d_%d", i, j)
			}
			c.SetBatch(items)

			keys := make([]string, 10)
			for j := 0; j < 10; j++ {
				keys[j] = fmt.Sprintf("batch_%d_%d", i, j)
			}
			c.GetBatch(keys)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 100; i++ {
			stats := c.Stats()
			if stats.Size > 10000 {
				errors <- fmt.Errorf("cache size exceeded capacity: %d", stats.Size)
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	stats := c.Stats()
	t.Logf("Final stats: Hits=%d, Misses=%d, Size=%d, HitRatio=%.2f%%",
		stats.Hits, stats.Misses, stats.Size, stats.HitRatio*100)

	if stats.Size > 10000 {
		t.Errorf("Final cache size exceeded capacity: %d", stats.Size)
	}
}

// performance with large values
func BenchmarkLargeValues(b *testing.B) {
	c := New(WithCapacity[string, []byte](1000))

	// Create a large value (1MB)
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, largeValue)
	}
}

// high contention scenarios
func BenchmarkHighContentionScenario(b *testing.B) {
	c := New(
		WithCapacity[string, string](100),
		WithThreadSafety[string, string](true),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("key_%d", rand.Intn(50))
			if rand.Float32() < 0.7 {
				c.Get(key)
			} else {
				c.Set(key, fmt.Sprintf("value_%d", rand.Intn(1000)))
			}
		}
	})
}

// memory allocation patterns
func BenchmarkMemoryAllocations(b *testing.B) {
	c := New(WithCapacity[string, string](1000))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, fmt.Sprintf("value_%d", i))
	}
}

// memory leaks over time
func BenchmarkMemoryLeaks(b *testing.B) {
	c := New(WithCapacity[string, string](1000))

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		c.Set(key, fmt.Sprintf("value_%d", i))
		if i%100 == 0 {
			c.Clear() // Periodic cleanup
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	memDiff := m2.Alloc - m1.Alloc
	b.ReportMetric(float64(memDiff)/1024, "KB-leaked")

	if memDiff > 1024*1024 { // 1MB threshold
		b.Errorf("Potential memory leak detected: %d bytes", memDiff)
	}
}

// garbage collection pressure
func BenchmarkGCPressure(b *testing.B) {
	c := New(WithCapacity[string, string](1000))

	var gcBefore, gcAfter runtime.MemStats
	runtime.ReadMemStats(&gcBefore)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%500)
		c.Set(key, fmt.Sprintf("value_%d", i))
		if i%100 == 0 {
			c.Delete(key)
		}
	}

	runtime.ReadMemStats(&gcAfter)
	gcCycles := gcAfter.NumGC - gcBefore.NumGC
	b.ReportMetric(float64(gcCycles), "gc-cycles")
}

// memory usage of different cache sizes
func BenchmarkMemoryEfficiency(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%d", size), func(b *testing.B) {
			var m1, m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			c := New(WithCapacity[string, string](size))

			// Fill cache to capacity
			for i := 0; i < size; i++ {
				c.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
			}

			runtime.GC()
			runtime.ReadMemStats(&m2)

			memPerItem := float64(m2.Alloc-m1.Alloc) / float64(size)
			b.ReportMetric(memPerItem, "bytes-per-item")
		})
	}
}
