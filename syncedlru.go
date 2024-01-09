package freelru

import (
	"sync"
	"time"
)

type SyncedLRU[K comparable, V any] struct {
	mu  sync.RWMutex
	lru *LRU[K, V]
}

var _ Cache[int, int] = (*SyncedLRU[int, int])(nil)

// SetLifetime sets the default lifetime of LRU elements.
// Lifetime 0 means "forever".
func (lru *SyncedLRU[K, V]) SetLifetime(lifetime time.Duration) {
	lru.mu.Lock()
	lru.lru.SetLifetime(lifetime)
	lru.mu.Unlock()
}

// SetOnEvict sets the OnEvict callback function.
// The onEvict function is called for each evicted lru entry.
func (lru *SyncedLRU[K, V]) SetOnEvict(onEvict OnEvictCallback[K, V]) {
	lru.mu.Lock()
	lru.lru.SetOnEvict(onEvict)
	lru.mu.Unlock()
}

// NewSynced creates a new thread-safe LRU hashmap with the given capacity.
func NewSynced[K comparable, V any](capacity uint32, hash HashKeyCallback[K]) (*SyncedLRU[K, V], error) {
	return NewSyncedWithSize[K, V](capacity, capacity, hash)
}

func NewSyncedWithSize[K comparable, V any](capacity, size uint32, hash HashKeyCallback[K]) (*SyncedLRU[K, V], error) {
	lru, err := NewWithSize[K, V](capacity, size, hash)
	if err != nil {
		return nil, err
	}
	return &SyncedLRU[K, V]{lru: lru}, nil
}

// Len returns the number of elements stored in the cache.
func (lru *SyncedLRU[K, V]) Len() (length int) {
	lru.mu.RLock()
	length = lru.lru.Len()
	lru.mu.RUnlock()

	return
}

// AddWithLifetime adds a key:value to the cache with a lifetime.
// Returns true, true if key was updated and eviction occurred.
func (lru *SyncedLRU[K, V]) AddWithLifetime(key K, value V, lifetime time.Duration) (evicted bool) {
	hash := lru.lru.hash(key)

	lru.mu.Lock()
	evicted = lru.lru.addWithLifetime(hash, key, value, lifetime)
	lru.mu.Unlock()

	return
}

// Add adds a key:value to the cache.
// Returns true, true if key was updated and eviction occurred.
func (lru *SyncedLRU[K, V]) Add(key K, value V) (evicted bool) {
	hash := lru.lru.hash(key)

	lru.mu.Lock()
	evicted = lru.lru.add(hash, key, value)
	lru.mu.Unlock()

	return
}

// Get looks up a key's value from the cache, setting it as the most
// recently used item.
func (lru *SyncedLRU[K, V]) Get(key K) (value V, ok bool) {
	hash := lru.lru.hash(key)

	lru.mu.Lock()
	value, ok = lru.lru.get(hash, key)
	lru.mu.Unlock()

	return
}

// Peek looks up a key's value from the cache, without changing its recent-ness.
func (lru *SyncedLRU[K, V]) Peek(key K) (value V, ok bool) {
	hash := lru.lru.hash(key)

	lru.mu.RLock()
	value, ok = lru.lru.peek(hash, key)
	lru.mu.RUnlock()

	return
}

// Contains checks for the existence of a key, without changing its recent-ness.
func (lru *SyncedLRU[K, V]) Contains(key K) (ok bool) {
	hash := lru.lru.hash(key)

	lru.mu.RLock()
	ok = lru.lru.contains(hash, key)
	lru.mu.RUnlock()

	return
}

// Remove removes the key from the cache.
// The return value indicates whether the key existed or not.
func (lru *SyncedLRU[K, V]) Remove(key K) (removed bool) {
	hash := lru.lru.hash(key)

	lru.mu.Lock()
	removed = lru.lru.remove(hash, key)
	lru.mu.Unlock()

	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (lru *SyncedLRU[K, V]) Keys() (keys []K) {
	lru.mu.RLock()
	keys = lru.lru.Keys()
	lru.mu.RUnlock()

	return
}

// Purge purges all data (key and value) from the LRU.
func (lru *SyncedLRU[K, V]) Purge() {
	lru.mu.Lock()
	lru.lru.Purge()
	lru.mu.Unlock()
}

// just used for debugging
func (lru *SyncedLRU[K, V]) dump() {
	lru.mu.RLock()
	lru.lru.dump()
	lru.mu.RUnlock()
}

func (lru *SyncedLRU[K, V]) PrintStats() {
	lru.mu.RLock()
	lru.lru.PrintStats()
	lru.mu.RUnlock()
}
