package storage

import (
	"testing"
	"time"
)

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage[string, string]()

	// Test Set and Get
	item := &Item[string]{Value: "value1"}
	storage.Set("key1", item)

	if retrievedItem, ok := storage.Get("key1"); !ok || retrievedItem.Value != "value1" {
		t.Errorf("Expected value1, got %v", retrievedItem)
	}

	// Test non-existent key
	if _, ok := storage.Get("nonexistent"); ok {
		t.Error("Expected false for non-existent key")
	}

	// Test Delete
	if !storage.Delete("key1") {
		t.Error("Expected true for successful delete")
	}

	if _, ok := storage.Get("key1"); ok {
		t.Error("Expected false after delete")
	}

	// Test Size
	item2 := &Item[string]{Value: "value2"}
	item3 := &Item[string]{Value: "value3"}
	storage.Set("key2", item2)
	storage.Set("key3", item3)

	if storage.Size() != 2 {
		t.Errorf("Expected size 2, got %d", storage.Size())
	}

	// Test Keys
	keys := storage.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Test Clear
	storage.Clear()
	if storage.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", storage.Size())
	}
}

func TestMemoryStorageWithTTL(t *testing.T) {
	storage := NewMemoryStorage[string, string]()

	// Test item with TTL
	item := &Item[string]{Value: "value1"}
	item.SetTTL(100 * time.Millisecond)
	storage.Set("key1", item)

	// Should be available immediately
	if retrievedItem, ok := storage.Get("key1"); !ok || retrievedItem.Value != "value1" {
		t.Errorf("Expected value1, got %v", retrievedItem)
	}

	// Should have TTL set
	if !item.HasTTL {
		t.Error("Expected item to have TTL")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	if _, ok := storage.Get("key1"); ok {
		t.Error("Expected false for expired key")
	}
}

func TestMemoryStorageCleanup(t *testing.T) {
	storage := NewMemoryStorage[string, string]()

	// Add items with different TTL
	item1 := &Item[string]{Value: "value1"}
	item1.SetTTL(-time.Hour) // Already expired (past time)

	item2 := &Item[string]{Value: "value2"}
	// item2 has no TTL (never expires)

	item3 := &Item[string]{Value: "value3"}
	item3.SetTTL(time.Hour) // Expires in 1 hour

	storage.Set("key1", item1)
	storage.Set("key2", item2)
	storage.Set("key3", item3)

	// 3 items before cleanup
	if storage.Size() != 3 {
		t.Errorf("Expected size 3, got %d", storage.Size())
	}

	// should remove 1 expired item
	removed := storage.CleanupExpired()
	if removed != 1 {
		t.Errorf("Expected 1 removed item, got %d", removed)
	}

	//  2 items after cleanup
	if storage.Size() != 2 {
		t.Errorf("Expected size 2 after cleanup, got %d", storage.Size())
	}
}

func TestMemoryStorageGenericTypes(t *testing.T) {
	// Test with different types
	intStorage := NewMemoryStorage[int, string]()
	intItem := &Item[string]{Value: "integer key"}
	intStorage.Set(42, intItem)

	if retrievedItem, ok := intStorage.Get(42); !ok || retrievedItem.Value != "integer key" {
		t.Errorf("Expected 'integer key', got %v", retrievedItem)
	}

	// Test with struct types
	type User struct {
		ID   int
		Name string
	}

	userStorage := NewMemoryStorage[string, User]()
	user := User{ID: 1, Name: "John"}
	userItem := &Item[User]{Value: user}
	userStorage.Set("user1", userItem)

	if retrievedItem, ok := userStorage.Get("user1"); !ok || retrievedItem.Value.Name != "John" {
		t.Errorf("Expected user John, got %v", retrievedItem)
	}
}

func TestMemoryStorageWithConfig(t *testing.T) {
	// Test with non-thread-safe storage
	storage := NewMemoryStorageWithConfig[string, string](10, false)

	item := &Item[string]{Value: "test"}
	storage.Set("key1", item)

	if retrievedItem, ok := storage.Get("key1"); !ok || retrievedItem.Value != "test" {
		t.Errorf("Expected test, got %v", retrievedItem)
	}

	// Reserve
	storage.Reserve(100)
	if storage.Size() != 1 {
		t.Errorf("Expected size 1 after reserve, got %d", storage.Size())
	}
}

func TestItemPool(t *testing.T) {
	pool := NewItemPool[string]()

	// Get item
	item := pool.Get()
	item.Value = "test"
	item.SetTTL(time.Hour)

	if item.Value != "test" || !item.HasTTL {
		t.Error("Expected item to be properly initialized")
	}

	// Put item
	pool.Put(item)

	// Item should be reset
	if item.Value != "" || item.HasTTL {
		t.Error("Expected item to be reset after returning to pool")
	}

	// another item from pool (should be reused)
	item2 := pool.Get()
	if item2.Value != "" || item2.HasTTL {
		t.Error("Expected clean item from pool")
	}
}

func TestItemSetTTL(t *testing.T) {
	item := &Item[string]{Value: "test"}

	item.SetTTL(time.Hour)
	if !item.HasTTL {
		t.Error("Expected item to have TTL")
	}

	if item.IsExpired() {
		t.Error("Expected item not to be expired")
	}

	// zero TTL
	item.SetTTL(0)
	if item.HasTTL {
		t.Error("Expected item not to have TTL")
	}

	// negative TTL (should be expired)
	item.SetTTL(-time.Hour)
	if !item.HasTTL {
		t.Error("Expected item to have TTL")
	}

	if !item.IsExpired() {
		t.Error("Expected item to be expired")
	}
}
