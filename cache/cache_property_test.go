package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestProperty_ValueConsistency demonstrates how property testing can verify
// that values are consistently stored and retrieved from the cache
func TestProperty_ValueConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use a small cache to force evictions
		cache := NewWithConfig(Config{
			MaxSize:    rapid.IntRange(2, 5).Draw(t, "cacheSize"),
			DefaultTTL: time.Duration(rapid.Int64Range(50, 200).Draw(t, "defaultTTLMs")) * time.Millisecond,
		})
		defer func() {
			cache.Stop()
			cache.Clear()
		}()

		numPairs := rapid.IntRange(5, 10).Draw(t, "numPairs")
		keys := make([]string, numPairs)
		values := make([]interface{}, numPairs)

		for i := 0; i < numPairs; i++ {
			keys[i] = rapid.StringN(1, 10, 10).Draw(t, "key"+string(rune('A'+i)))
			values[i] = rapid.OneOf(
				rapid.Int().AsAny(),
				rapid.String().AsAny(),
				rapid.SliceOf(rapid.Int()).AsAny(),
				rapid.Bool().AsAny(), // Add more types to increase complexity
			).Draw(t, "value"+string(rune('A'+i)))
		}

		// Track what should be in the cache
		expected := make(map[string]interface{})

		// Property 1: Values should be retrievable immediately after setting
		for i := 0; i < numPairs; i++ {
			cache.Set(keys[i], values[i], 0)
			expected[keys[i]] = values[i]

			// This might fail due to immediate LRU eviction if cache is full
			got, exists := cache.Get(keys[i])
			if exists {
				require.Equal(t, values[i], got,
					"Retrieved value should match stored value for key %s", keys[i])
			} else {
				// If it doesn't exist, it must be because we exceeded cache size
				require.Greater(t, len(expected), cache.config.MaxSize,
					"Missing value must be due to cache size limit")
			}
		}

		// Property 2: Test time-based eviction
		timeJump := rapid.Int64Range(0, 300).Draw(t, "timeJumpMs")
		time.Sleep(time.Duration(timeJump) * time.Millisecond)

		for key, expectedVal := range expected {
			got, exists := cache.Get(key)

			// If the time jump exceeded TTL, the item should be gone
			if timeJump > int64(cache.config.DefaultTTL/time.Millisecond) {
				require.False(t, exists,
					"Key %s should be expired after %dms (TTL: %dms)",
					key, timeJump, cache.config.DefaultTTL/time.Millisecond)
			} else if exists {
				require.Equal(t, expectedVal, got,
					"Value for key %s should match after %dms", key, timeJump)
			}
		}

		// Property 3: Test LRU eviction under load
		numOps := rapid.IntRange(10, 20).Draw(t, "numOps")
		for i := 0; i < numOps; i++ {
			opKind := rapid.IntRange(0, 2).Draw(t, "opKind")
			keyIdx := rapid.IntRange(0, numPairs-1).Draw(t, "keyIdx")

			switch opKind {
			case 0: // Get
				got, exists := cache.Get(keys[keyIdx])
				if exists {
					require.Equal(t, expected[keys[keyIdx]], got,
						"Retrieved value should match for key %s", keys[keyIdx])
				}
			case 1: // Set
				newVal := rapid.String().Draw(t, "newValue")
				cache.Set(keys[keyIdx], newVal, 0)
				expected[keys[keyIdx]] = newVal
			case 2: // Delete
				cache.Delete(keys[keyIdx])
				delete(expected, keys[keyIdx])
			}

			// Verify cache size constraint
			require.LessOrEqual(t, cache.Len(), cache.config.MaxSize,
				"Cache size %d should not exceed max size %d",
				cache.Len(), cache.config.MaxSize)
		}
	})
}

// TestProperty_DeliberateFailure demonstrates how Rapid handles and reports test failures
func TestProperty_DeliberateFailure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create a cache that's deliberately too small
		cache := NewWithConfig(Config{
			MaxSize:    2,
			DefaultTTL: time.Hour,
		})
		defer func() {
			cache.Stop()
			cache.Clear()
		}()

		// Always generate exactly 3 pairs - one more than cache size
		const numPairs = 3
		keys := make([]string, numPairs)
		values := make([]string, numPairs)
		for i := 0; i < numPairs; i++ {
			keys[i] = rapid.StringN(1, 10, 10).Draw(t, "key"+string(rune('A'+i)))
			values[i] = "value" + string(rune('A'+i))
		}

		for i := 0; i < numPairs; i++ {
			cache.Set(keys[i], values[i], 0)
		}

		// This will fail because we're asserting all values must exist
		// even though the cache size is too small
		for i := 0; i < numPairs; i++ {
			got, exists := cache.Get(keys[i])

			// This will never hold because the cache is too small
			require.True(t, exists,
				"Key %s should exist (i=%d, cache size=%d, numPairs=%d)",
				keys[i], i, cache.config.MaxSize, numPairs)

			if exists {
				require.Equal(t, values[i], got,
					"Value mismatch for key %s", keys[i])
			}
		}
	})
}
