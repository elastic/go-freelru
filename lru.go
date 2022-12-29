// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package go_freelru

import (
	"errors"
)

// OnEvictCallback is the function type for Config.OnEvict.
type OnEvictCallback[K comparable, V any] func(K, V)

// HashKeyCallback is the function that creates a hash from the passed key.
type HashKeyCallback[K comparable] func(K) uint32

// Config is the type for the LRU configuration passed to New().
type Config[K comparable, V any] struct {
	// Capacity is the maximal capacity of the LRU cache before eviction takes place.
	Capacity uint32

	// OnEvict is called for every eviction.
	OnEvict OnEvictCallback[K, V]

	// HashKey is called for whenever the hash of a key is needed.
	HashKey HashKeyCallback[K]

	// MemoryFactor is an over-provisioned factor for the hashtable size to reduce
	// the probability of collisions, as these are relatve expensive in terms of CPU usage.
	// The default is 1.25 (25% over-committed).
	MemoryFactor float64
}

// DefaultConfig returns a default configuration for New().
func DefaultConfig[K comparable, V any]() Config[K, V] {
	return Config[K, V]{
		Capacity:     128,
		OnEvict:      nil,
		MemoryFactor: 1.25,
	}
}

type element[K comparable, V any] struct {
	key   K
	value V

	// next and prev are indexes in the doubly-linked list of elements.
	// To simplify the implementation, internally a list l is implemented
	// as a ring, such that &l.latest.prev is last element and
	// &l.last.next is the latest element.
	next, prev uint32

	// Marks the element as free
	free bool
}

// LRU implements a non-thread safe fixed size LRU cache.
type LRU[K comparable, V any] struct {
	elements []element[K, V]
	onEvict  OnEvictCallback[K, V]
	hash     HashKeyCallback[K]

	head uint32 // index of latest element in cache
	len  uint32 // number of elements in the cache
	cap  uint32 // capacity of cache
	size uint32 // size of the element array (25% larger than cap)
}

// New constructs an LRU with the given capacity of elements.
// The onEvict function is called for each evicted cache entry.
func New[K comparable, V any](cap uint32, onEvict OnEvictCallback[K, V],
	hash HashKeyCallback[K]) (*LRU[K, V], error) {
	cfg := DefaultConfig[K, V]()
	cfg.Capacity = cap
	cfg.OnEvict = onEvict
	cfg.HashKey = hash
	return NewWithConfig(cfg)
}

// NewWithConfig constructs an LRU from the given configuration.
func NewWithConfig[K comparable, V any](cfg Config[K, V]) (*LRU[K, V], error) {
	if cfg.Capacity == 0 {
		return nil, errors.New("Capacity must be positive")
	}
	if cfg.MemoryFactor < 1.0 {
		return nil, errors.New("MemoryFactor must be >= 1.0")
	}
	if cfg.HashKey == nil {
		return nil, errors.New("HashKey must be set")
	}

	// The hashtable size is over-provisioned by 25% to reduce collisions
	// as collisions are relatively expensive.
	size := uint32(float64(cfg.Capacity)*cfg.MemoryFactor) + 1
	lru := &LRU[K, V]{
		cap:      cfg.Capacity,
		size:     size,
		elements: make([]element[K, V], size),
		onEvict:  cfg.OnEvict,
		hash:     cfg.HashKey,
	}

	// Mark all slots as free.
	for i := range lru.elements {
		lru.elements[i].free = true
	}

	return lru, nil
}

// keyToPos converts a key into a position in the elements array.
func (lru *LRU[K, V]) keyToPos(key K) uint32 {
	return lru.hash(key) % lru.size
}

// nextPos avoids the costly modulo operation.
func (lru *LRU[K, V]) nextPos(pos uint32) uint32 {
	if pos+1 != lru.size {
		return pos + 1
	}
	return 0
}

// setHead links the element as the head into the list.
func (lru *LRU[K, V]) setHead(pos uint32) {
	if pos == lru.head {
		panic(pos)
	}

	lru.elements[pos].prev = lru.head
	lru.elements[pos].next = lru.elements[lru.head].next
	lru.elements[lru.elements[lru.head].next].prev = pos
	lru.elements[lru.head].next = pos
	lru.head = pos
}

// unlink removes the element from the list.
func (lru *LRU[K, V]) unlink(pos uint32) {
	lru.elements[lru.elements[pos].prev].next = lru.elements[pos].next
	lru.elements[lru.elements[pos].next].prev = lru.elements[pos].prev
}

// evict evicts the element at the given position.
func (lru *LRU[K, V]) evict(pos uint32) {
	if pos == lru.head {
		lru.head = lru.elements[pos].prev
	}

	lru.unlink(pos)
	lru.len--

	lru.elements[pos].free = true

	if lru.onEvict != nil {
		// The following code avoids conflicts in case the onEvict callback
		// inserts the key and value into the cache.
		key := lru.elements[pos].key
		value := lru.elements[pos].value
		lru.onEvict(key, value)
	}
}

// insert stores the k/v at pos.
// It updates the head to point to this position.
func (lru *LRU[K, V]) insert(pos uint32, key K, value V) {
	lru.elements[pos].key = key
	lru.elements[pos].value = value
	lru.elements[pos].free = false

	if lru.len == 0 {
		lru.elements[pos].prev = pos
		lru.elements[pos].next = pos
		lru.head = pos
	} else if pos != lru.head {
		lru.setHead(pos)
	}
	lru.len++
}

// shiftKeys rearranges the elements back into a defined layout
// after the entry at currentPosition has been removed.
func (lru *LRU[K, V]) shiftKeys(currentPosition uint32) {
	for {
		freeSlot := currentPosition
		currentPosition = lru.nextPos(currentPosition)
		for {
			if lru.elements[currentPosition].free {
				lru.elements[freeSlot].free = true
				return
			}
			currentKeySlot := lru.keyToPos(lru.elements[currentPosition].key)
			if freeSlot <= currentPosition {
				if freeSlot >= currentKeySlot || currentKeySlot > currentPosition {
					break
				}
			} else {
				if freeSlot >= currentKeySlot && currentKeySlot > currentPosition {
					break
				}
			}
			currentPosition = lru.nextPos(currentPosition)
		}
		lru.elements[freeSlot] = lru.elements[currentPosition]
		lru.elements[lru.elements[currentPosition].prev].next = freeSlot
		lru.elements[lru.elements[currentPosition].next].prev = freeSlot
		if currentPosition == lru.head {
			lru.head = freeSlot
		}
	}
}

// Len returns the number of elements stored in the cache.
func (lru *LRU[K, V]) Len() int {
	return int(lru.len)
}

// Add adds a key:value to the cache.
// Returns true, true if key was updated and eviction occurred.
func (lru *LRU[K, V]) Add(key K, value V) (evicted bool) {
	startPosition := lru.keyToPos(key)
	for pos := startPosition; ; pos = lru.nextPos(pos) {
		if lru.elements[pos].free {
			// lru.len should never exceed lru.cap
			if lru.len == lru.cap {
				// evict oldest entry
				tailPos := lru.elements[lru.head].next
				lru.evict(tailPos)
				lru.shiftKeys(tailPos)
				evicted = true
				// After evict and shiftKeys have taken place, the structure
				// of the LRU has changed and if we insert at this point we
				// could be breaking the invariant: There's no guarantee that
				// positions that this search previously skipped because they
				// were invalid continue to hold the same keys.
				// break triggers a new search starting from startPosition.
				break
			}

			lru.insert(pos, key, value)
			return
		}

		if key == lru.elements[pos].key {
			// Key exists, replace the value and update element to be the head element.
			lru.elements[pos].value = value

			if pos != lru.head {
				lru.unlink(pos)
				lru.setHead(pos)
			}

			return false
		}
	}

	for pos := startPosition; ; pos = lru.nextPos(pos) {
		if !lru.elements[pos].free {
			continue
		}

		lru.insert(pos, key, value)
		return
	}
}

// Get looks up a key's value from the cache, setting it as the most
// recently used item.
func (lru *LRU[K, V]) Get(key K) (value V, ok bool) {
	for pos := lru.keyToPos(key); !lru.elements[pos].free; pos = lru.nextPos(pos) {
		if key == lru.elements[pos].key {
			if pos != lru.head {
				lru.unlink(pos)
				lru.setHead(pos)
			}
			return lru.elements[pos].value, true
		}
	}

	// We cannot use nil for an empty type
	var v0 V
	return v0, false
}

// Peek looks up a key's value from the cache, without changing its recent-ness.
func (lru *LRU[K, V]) Peek(key K) (value V, ok bool) {
	for pos := lru.keyToPos(key); !lru.elements[pos].free; pos = lru.nextPos(pos) {
		if key == lru.elements[pos].key {
			return lru.elements[pos].value, true
		}
	}

	// We cannot use nil for an empty type
	var v0 V
	return v0, false
}

// Contains checks for the existence of a key, without changing its recent-ness.
func (lru *LRU[K, V]) Contains(key K) (ok bool) {
	_, ok = lru.Peek(key)
	return
}

// Remove removes the key from the cache.
// The return value indicates whether the key existed or not.
func (lru *LRU[K, V]) Remove(key K) (removed bool) {
	for pos := lru.keyToPos(key); !lru.elements[pos].free; pos = lru.nextPos(pos) {
		if key == lru.elements[pos].key {
			// Key exists, update element to be the head element.
			lru.evict(pos)
			lru.shiftKeys(pos)
			return true
		}
	}

	return false
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (lru *LRU[K, V]) Keys() []K {
	keys := make([]K, 0, lru.len)
	pos := lru.elements[lru.head].next
	for i := uint32(0); i < lru.len; i++ {
		keys = append(keys, lru.elements[pos].key)
		pos = lru.elements[pos].next
	}
	return keys
}
