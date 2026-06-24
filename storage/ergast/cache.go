package ergast

import (
	"racebot-vk/models"
	"sync"
	"time"
)

type cacheEntry struct {
	data      models.Object
	expiresAt time.Time
}

type cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

func newCache(ttl time.Duration) *cache {
	return &cache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

func (c *cache) get(key string) (models.Object, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return models.Object{}, false
	}

	if time.Now().After(entry.expiresAt) {
		return models.Object{}, false
	}

	return entry.data, true
}

func (c *cache) set(key string, data models.Object) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}
