package freelru

import (
	"sync"
	"time"
)

type SyncedLRU[K comparable, V any] struct {
	mu  sync.RWMutex
	lru *LRU[K, V]
}

// SetLifetime sets the default lifetime of LRU elements.
// Lifetime 0 means "forever".
func (lru *SyncedLRU[K, V]) SetLifetime(lifetime time.Duration) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	lru.lru.SetLifetime(lifetime)
}

// SetOnEvict sets the OnEvict callback function.
// The onEvict function is called for each evicted lru entry.
func (lru *SyncedLRU[K, V]) SetOnEvict(onEvict OnEvictCallback[K, V]) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	lru.lru.SetOnEvict(onEvict)
}

// NewSynced creates a new thread-safe LRU hashmap with the given capacity.
func NewSynced[K comparable, V any](cap uint32, hash HashKeyCallback[K]) (*SyncedLRU[K, V], error) {
	return NewSyncedWithSize[K, V](cap, cap, hash)
}

func NewSyncedWithSize[K comparable, V any](cap, size uint32, hash HashKeyCallback[K]) (*SyncedLRU[K, V], error) {
	lru, err := NewWithSize[K, V](cap, size, hash)
	if err != nil {
		return nil, err
	}
	return &SyncedLRU[K, V]{lru: lru}, nil
}

// Len returns the number of elements stored in the cache.
func (lru *SyncedLRU[K, V]) Len() int {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	return lru.lru.Len()
}

// AddWithLifetime adds a key:value to the cache with a lifetime.
// Returns true, true if key was updated and eviction occurred.
func (lru *SyncedLRU[K, V]) AddWithLifetime(key K, value V, lifetime time.Duration) (evicted bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.lru.AddWithLifetime(key, value, lifetime)
}

// Add adds a key:value to the cache.
// Returns true, true if key was updated and eviction occurred.
func (lru *SyncedLRU[K, V]) Add(key K, value V) (evicted bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.lru.AddWithLifetime(key, value, lru.lru.lifetime)
}

// Get looks up a key's value from the cache, setting it as the most
// recently used item.
func (lru *SyncedLRU[K, V]) Get(key K) (value V, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	value, ok = lru.lru.Get(key)
	return
}

// Peek looks up a key's value from the cache, without changing its recent-ness.
func (lru *SyncedLRU[K, V]) Peek(key K) (value V, ok bool) {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	return lru.lru.Peek(key)
}

// Contains checks for the existence of a key, without changing its recent-ness.
func (lru *SyncedLRU[K, V]) Contains(key K) (ok bool) {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	return lru.lru.Contains(key)
}

// Remove removes the key from the cache.
// The return value indicates whether the key existed or not.
func (lru *SyncedLRU[K, V]) Remove(key K) (removed bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.lru.Remove(key)
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (lru *SyncedLRU[K, V]) Keys() []K {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	return lru.lru.Keys()
}

// Purge purges all data (key and value) from the LRU.
func (lru *SyncedLRU[K, V]) Purge() {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	lru.lru.Purge()
}

// just used for debugging
func (lru *SyncedLRU[K, V]) dump() {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	lru.lru.dump()
}

func (lru *SyncedLRU[K, V]) PrintStats() {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	lru.lru.PrintStats()
}
