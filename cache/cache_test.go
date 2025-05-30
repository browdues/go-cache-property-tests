package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCache_SizeLimit(t *testing.T) {
	config := Config{
		MaxSize:    2,
		DefaultTTL: time.Hour,
	}
	cache := NewWithConfig(config)
	defer func() {
		cache.Stop()
		cache.Clear()
	}()

	cache.Set("1", "one", 0)
	cache.Set("2", "two", 0)
	cache.Set("3", "three", 0)

	// Cache size
	require.Equal(t, 2, cache.Len())

	// LRU eviction
	_, exists := cache.Get("1")
	require.False(t, exists, "Oldest item should be evicted")

	val2, exists := cache.Get("2")
	require.True(t, exists)
	require.Equal(t, "two", val2)

	val3, exists := cache.Get("3")
	require.True(t, exists)
	require.Equal(t, "three", val3)

	// Access item 2 (thus making it most recently used)
	cache.Get("2")

	// Add new item (should evict item 3)
	cache.Set("4", "four", 0)

	// Check LRU state
	val2, exists = cache.Get("2")
	require.True(t, exists)
	require.Equal(t, "two", val2)

	val4, exists := cache.Get("4")
	require.True(t, exists)
	require.Equal(t, "four", val4)

	_, exists = cache.Get("3")
	require.False(t, exists)
}

func TestCache_DefaultTTL(t *testing.T) {
	config := Config{
		MaxSize:    10,
		DefaultTTL: 50 * time.Millisecond,
	}
	cache := NewWithConfig(config)
	defer func() {
		cache.Stop()
		cache.Clear()
	}()

	cache.Set("key", "value", 0)

	// Before expiration
	val, exists := cache.Get("key")
	require.True(t, exists)
	require.Equal(t, "value", val)

	time.Sleep(60 * time.Millisecond)

	// After expiration
	_, exists = cache.Get("key")
	require.False(t, exists)
}

func TestCache_ExplicitExpiration(t *testing.T) {
	cache := New()
	defer func() {
		cache.Stop()
		cache.Clear()
	}()

	cache.Set("key", "value", 50*time.Millisecond)

	// Before expiration
	val, exists := cache.Get("key")
	require.True(t, exists)
	require.Equal(t, "value", val)

	time.Sleep(60 * time.Millisecond)

	// After expiration
	_, exists = cache.Get("key")
	require.False(t, exists)
}

func TestCache_ClearAndLen(t *testing.T) {
	cache := New()
	defer cache.Stop()

	cache.Set("1", "one", 0)
	cache.Set("2", "two", 0)
	require.Equal(t, 2, cache.Len())

	cache.Clear()
	require.Equal(t, 0, cache.Len())

	cache.Set("3", "three", 0)
	require.Equal(t, 1, cache.Len())
}

func TestCache_Delete(t *testing.T) {
	cache := New()
	defer cache.Stop()

	cache.Set("key", "value", 0)
	require.Equal(t, 1, cache.Len())

	cache.Delete("key")
	require.Equal(t, 0, cache.Len())

	_, exists := cache.Get("key")
	require.False(t, exists)

	cache.Delete("nonexistent")
}
