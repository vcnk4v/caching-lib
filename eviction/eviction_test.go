package eviction

import (
	"testing"
)

func TestLRUEviction(t *testing.T) {
	policy := NewLRU[string](3)

	policy.Access("key1")
	policy.Access("key2")
	policy.Access("key3")

	// Access key1 to make it recently used
	policy.Access("key1")

	// Should have 3 items
	if policy.Size() != 3 {
		t.Errorf("Expected size 3, got %d", policy.Size())
	}

	// Add another item, should evict key2 (least recently used)
	policy.Access("key4")
	evicted, hasEvicted := policy.Evict()

	if !hasEvicted {
		t.Error("Expected to have an item to evict")
	}

	if evicted != "key2" {
		t.Errorf("Expected key2 to be evicted, got %s", evicted)
	}
}

func TestLRUEvictionOrder(t *testing.T) {
	policy := NewLRU[string](2)

	// Add items in order
	policy.Access("first")
	policy.Access("second")

	// Access first to make it most recent
	policy.Access("first")

	// Add third item
	policy.Access("third")

	// Should evict "second" (least recently used)
	evicted, hasEvicted := policy.Evict()
	if !hasEvicted || evicted != "second" {
		t.Errorf("Expected 'second' to be evicted, got %s", evicted)
	}
}

func TestFIFOEviction(t *testing.T) {
	policy := NewFIFO[string](3)

	// Add items
	policy.Access("key1")
	policy.Access("key2")
	policy.Access("key3")

	// Should have 3 items
	if policy.Size() != 3 {
		t.Errorf("Expected size 3, got %d", policy.Size())
	}

	// Evict should return key1 (first in)
	evicted, hasEvicted := policy.Evict()
	if !hasEvicted || evicted != "key1" {
		t.Errorf("Expected key1 to be evicted, got %s", evicted)
	}
}

func TestFIFOEvictionOrder(t *testing.T) {
	policy := NewFIFO[string](2)

	// Add items in order
	policy.Access("first")
	policy.Access("second")

	// Access first again (should not change order in FIFO)
	policy.Access("first")

	// Add third item
	policy.Access("third")

	// Should evict "first" (first in)
	evicted, hasEvicted := policy.Evict()
	if !hasEvicted || evicted != "first" {
		t.Errorf("Expected 'first' to be evicted, got %s", evicted)
	}
}

func TestLIFOEviction(t *testing.T) {
	policy := NewLIFO[string](3)

	// Add items
	policy.Access("key1")
	policy.Access("key2")
	policy.Access("key3")

	// Should have 3 items
	if policy.Size() != 3 {
		t.Errorf("Expected size 3, got %d", policy.Size())
	}

	// Evict should return key3 (last in)
	evicted, hasEvicted := policy.Evict()
	if !hasEvicted || evicted != "key3" {
		t.Errorf("Expected key3 to be evicted, got %s", evicted)
	}
}

func TestLIFOEvictionOrder(t *testing.T) {
	policy := NewLIFO[string](2)

	// Add items in order
	policy.Access("first")
	policy.Access("second")

	// Access first again (should not change order in LIFO)
	policy.Access("first")

	// Add third item
	policy.Access("third")

	// Should evict "third" (last in)
	evicted, hasEvicted := policy.Evict()
	if !hasEvicted || evicted != "third" {
		t.Errorf("Expected 'third' to be evicted, got %s", evicted)
	}
}

func TestPolicyRemove(t *testing.T) {
	policy := NewLRU[string](3)

	policy.Access("key1")
	policy.Access("key2")

	if policy.Size() != 2 {
		t.Errorf("Expected size 2, got %d", policy.Size())
	}

	policy.Remove("key1")

	if policy.Size() != 1 {
		t.Errorf("Expected size 1, got %d", policy.Size())
	}

	// Try to remove non-existent key
	policy.Remove("nonexistent")

	if policy.Size() != 1 {
		t.Errorf("Expected size 1 after removing non-existent key, got %d", policy.Size())
	}
}

func TestPolicyClear(t *testing.T) {
	policy := NewLRU[string](3)

	policy.Access("key1")
	policy.Access("key2")

	policy.Clear()

	if policy.Size() != 0 {
		t.Errorf("Expected size 0, got %d", policy.Size())
	}

	// Should have nothing to evict after clear
	if _, hasEvicted := policy.Evict(); hasEvicted {
		t.Error("Expected no item to evict after clear")
	}
}

func TestPolicyEmptyEviction(t *testing.T) {
	policy := NewLRU[string](3)

	// Should have nothing to evict when empty
	if _, hasEvicted := policy.Evict(); hasEvicted {
		t.Error("Expected no item to evict when empty")
	}
}

func TestPolicyGenericTypes(t *testing.T) {
	// Test with integer keys
	intPolicy := NewLRU[int](3)
	intPolicy.Access(1)
	intPolicy.Access(2)
	intPolicy.Access(3)

	if intPolicy.Size() != 3 {
		t.Errorf("Expected size 3, got %d", intPolicy.Size())
	}

	evicted, hasEvicted := intPolicy.Evict()
	if !hasEvicted || evicted != 1 {
		t.Errorf("Expected 1 to be evicted, got %d", evicted)
	}

	// Test with struct keys
	type User struct {
		ID   int
		Name string
	}

	userPolicy := NewLRU[User](2)
	user1 := User{ID: 1, Name: "John"}
	user2 := User{ID: 2, Name: "Jane"}

	userPolicy.Access(user1)
	userPolicy.Access(user2)

	if userPolicy.Size() != 2 {
		t.Errorf("Expected size 2, got %d", userPolicy.Size())
	}

	evictedUser, hasEvicted := userPolicy.Evict()
	if !hasEvicted || evictedUser != user1 {
		t.Errorf("Expected user1 to be evicted, got %v", evictedUser)
	}
}

func TestPolicyCapacityValidation(t *testing.T) {
	// Test with zero capacity
	policy := NewLRU[string](0)
	policy.Access("key1")

	if policy.Size() != 1 {
		t.Errorf("Expected size 1 even with zero capacity, got %d", policy.Size())
	}

	// Test with negative capacity
	policy2 := NewLRU[string](-1)
	policy2.Access("key1")

	if policy2.Size() != 1 {
		t.Errorf("Expected size 1 even with negative capacity, got %d", policy2.Size())
	}
}
