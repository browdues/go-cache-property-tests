package cache

// This is an LRU cache with a sync.Map for concurrent access to items.
// It was created as an exmmple to illustrate property-based testing.

import (
	"container/list"
	"sync"
	"time"
)

// Config holds cache configuration options
type Config struct {
	MaxSize int

	// If 0, items without explicit expiration won't expire
	DefaultTTL time.Duration
}

// DefaultConfig returns the default cache configuration
func DefaultConfig() Config {
	return Config{
		MaxSize:    1000,
		DefaultTTL: 24 * time.Hour,
	}
}

// Item represents a cache entry with value and expiration
type Item struct {
	Value      any
	Expiration int64
}

// Cache provides a thread-safe cache with expiration and LRU eviction
type Cache struct {
	items  sync.Map
	lru    *list.List
	keyMap map[string]*list.Element
	lock   sync.Mutex
	config Config
	done   chan struct{}
}

func New() *Cache {
	return NewWithConfig(DefaultConfig())
}

func NewWithConfig(config Config) *Cache {
	cache := &Cache{
		lru:    list.New(),
		keyMap: make(map[string]*list.Element),
		config: config,
		done:   make(chan struct{}),
	}
	go cache.janitor()
	return cache
}

// Set adds a key-value pair to the cache with optional expiration
func (c *Cache) Set(key string, value any, duration time.Duration) {
	var exp int64
	if duration > 0 {
		exp = time.Now().Add(duration).UnixNano()
	} else if c.config.DefaultTTL > 0 {
		exp = time.Now().Add(c.config.DefaultTTL).UnixNano()
	}

	c.lock.Lock()
	if elem, exists := c.keyMap[key]; exists {
		c.lru.Remove(elem)
		delete(c.keyMap, key)
	}
	elem := c.lru.PushFront(key)
	c.keyMap[key] = elem

	if c.config.MaxSize > 0 && c.lru.Len() > c.config.MaxSize {
		if back := c.lru.Back(); back != nil {
			evictKey := back.Value.(string)
			c.lru.Remove(back)
			delete(c.keyMap, evictKey)
			c.items.Delete(evictKey)
		}
	}
	c.lock.Unlock()

	c.items.Store(key, Item{
		Value:      value,
		Expiration: exp,
	})
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (any, bool) {
	itemObj, exists := c.items.Load(key)
	if !exists {
		return nil, false
	}

	item := itemObj.(Item)

	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		c.Delete(key) // This handles LRU and sync.Map deletion
		return nil, false
	}

	c.lock.Lock()
	elem, lruExists := c.keyMap[key]
	if lruExists {
		c.lru.MoveToFront(elem)
	}
	c.lock.Unlock()

	// If the item was found in items but not in the LRU map (meaning it was
	// concurrently evicted/deleted after items.Load but before lock was acquired),
	// consider it not found.
	if !lruExists {
		return nil, false
	}

	return item.Value, true
}

// Delete removes a key from the cache
func (c *Cache) Delete(key string) {
	c.lock.Lock()
	if elem, exists := c.keyMap[key]; exists {
		c.lru.Remove(elem)
		delete(c.keyMap, key)
	}
	c.lock.Unlock()

	c.items.Delete(key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.lock.Lock()
	c.lru = list.New()
	c.keyMap = make(map[string]*list.Element)
	c.lock.Unlock()

	c.items = sync.Map{}
}

// Len returns the current number of items in the cache
func (c *Cache) Len() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Len()
}

// Stop cleanly shuts down the cache and stops the janitor
func (c *Cache) Stop() {
	close(c.done)
}

// janitor periodically cleans up expired items
func (c *Cache) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			c.items.Range(func(key, value any) bool {
				item := value.(Item)
				if item.Expiration > 0 && now > item.Expiration {
					c.Delete(key.(string))
				}
				return true
			})
		case <-c.done:
			return
		}
	}
}
