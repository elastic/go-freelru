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
	"fmt"
	"math"
	"math/bits"
	"time"
)

// OnEvictCallback is the function type for Config.OnEvict.
type OnEvictCallback[K comparable, V any] func(K, V)

// HashKeyCallback is the function that creates a hash from the passed key.
type HashKeyCallback[K comparable] func(K) uint32

type element[K comparable, V any] struct {
	key   K
	value V

	// bucketNext and bucketPrev are indexes in the space-dimension doubly-linked list of elements.
	// That is to add/remove items to the collision bucket without re-allocations and with O(1)
	// complexity.
	// To simplify the implementation, internally a list l is implemented
	// as a ring, such that &l.latest.prev is last element and
	// &l.last.next is the latest element.
	nextBucket, prevBucket uint32

	// bucketPos is the bucket that an element belongs to.
	bucketPos uint32

	// next and prev are indexes in the time-dimension doubly-linked list of elements.
	// To simplify the implementation, internally a list l is implemented
	// as a ring, such that &l.latest.prev is last element and
	// &l.last.next is the latest element.
	next, prev uint32

	// expire is the point in time when the element expires.
	// Its value is Unix milliseconds since epoch.
	expire int64
}

const emptyBucket = math.MaxUint32

// LRU implements a non-thread safe fixed size LRU cache.
type LRU[K comparable, V any] struct {
	buckets  []uint32 // contains positions of bucket lists or 'emptyBucket'
	elements []element[K, V]
	onEvict  OnEvictCallback[K, V]
	hash     HashKeyCallback[K]
	lifetime time.Duration

	// used for element clearing after removal or expiration
	emptyKey   K
	emptyValue V

	head uint32 // index of the newest element in the cache
	len  uint32 // current number of elements in the cache
	cap  uint32 // max number of elements in the cache
	size uint32 // size of the element array (X% larger than cap)
	mask uint32 // bitmask to avoid the costly idiv in keyToPos() if size is a 2^n value

	collisions uint64
	inserts    uint64
	evictions  uint64
	removals   uint64
}

// SetLifetime sets the default lifetime of LRU elements.
// Lifetime 0 means "forever".
func (lru *LRU[K, V]) SetLifetime(lifetime time.Duration) {
	lru.lifetime = lifetime
}

// SetOnEvict sets the OnEvict callback function.
// The onEvict function is called for each evicted lru entry.
func (lru *LRU[K, V]) SetOnEvict(onEvict OnEvictCallback[K, V]) {
	lru.onEvict = onEvict
}

// New constructs an LRU with the given capacity of elements.
// The hash function calculates a hash value from the keys.
func New[K comparable, V any](cap uint32, hash HashKeyCallback[K]) (*LRU[K, V], error) {
	return NewWithSize[K, V](cap, cap, hash)
}

// NewWithSize constructs an LRU with the given capacity and size.
// The hash function calculates a hash value from the keys.
// A size greater than the capacity increases memory consumption and decreases the CPU consumption
// by reducing the chance of collisions.
// Size must not be lower than the capacity.
func NewWithSize[K comparable, V any](cap, size uint32, hash HashKeyCallback[K]) (
	*LRU[K, V], error) {
	if cap == 0 {
		return nil, errors.New("capacity must be positive")
	}
	if size == emptyBucket {
		return nil, fmt.Errorf("size must not be %#X", size)
	}
	if size < cap {
		return nil, fmt.Errorf("size (%d) is smaller than capacity (%d)", size, cap)
	}
	if hash == nil {
		return nil, errors.New("hash function must be set")
	}

	lru := &LRU[K, V]{
		cap:      cap,
		size:     cap,
		hash:     hash,
		buckets:  make([]uint32, size),
		elements: make([]element[K, V], size),
	}

	// If the size is 2^N, we can avoid costly divisions.
	if bits.OnesCount32(lru.size) == 1 {
		lru.mask = lru.size - 1
	}

	// Mark all slots as free.
	for i := range lru.buckets {
		lru.buckets[i] = emptyBucket
	}

	return lru, nil
}

// keyToPos converts a key into a position in the elements array.
func (lru *LRU[K, V]) keyToBucketPos(key K) uint32 {
	if lru.mask != 0 {
		return lru.hash(key) & lru.mask
	}
	return lru.hash(key) % lru.size
}

// keyToPos converts a key into a position in the elements array.
func (lru *LRU[K, V]) keyToPos(key K) (bucketPos, elemPos uint32) {
	bucketPos = lru.keyToBucketPos(key)
	elemPos = lru.buckets[bucketPos]
	return
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

// unlinkElement removes the element from the elements list.
func (lru *LRU[K, V]) unlinkElement(pos uint32) {
	lru.elements[lru.elements[pos].prev].next = lru.elements[pos].next
	lru.elements[lru.elements[pos].next].prev = lru.elements[pos].prev
}

// unlinkBucket removes the element from the buckets list.
func (lru *LRU[K, V]) unlinkBucket(pos uint32) {
	prevBucket := lru.elements[pos].prevBucket
	nextBucket := lru.elements[pos].nextBucket
	if prevBucket == nextBucket && prevBucket == pos {
		// The element references itself, so it's the only bucket entry
		lru.buckets[lru.elements[pos].bucketPos] = emptyBucket
		return
	}
	lru.elements[prevBucket].nextBucket = nextBucket
	lru.elements[nextBucket].prevBucket = prevBucket
	lru.buckets[lru.elements[pos].bucketPos] = nextBucket
}

// evict evicts the element at the given position.
func (lru *LRU[K, V]) evict(pos uint32) {
	if pos == lru.head {
		lru.head = lru.elements[pos].prev
	}

	lru.unlinkElement(pos)
	lru.unlinkBucket(pos)
	lru.len--
	lru.evictions++

	if lru.onEvict != nil {
		// Save k/v for the eviction function.
		key := lru.elements[pos].key
		value := lru.elements[pos].value
		lru.onEvict(key, value)
	}
}

// Move element from position old to new.
// That avoids 'gaps' and new elements can always be simply appended.
func (lru *LRU[K, V]) move(new, old uint32) {
	if new == old {
		return
	}
	if old == lru.head {
		lru.head = new
	}

	prev := lru.elements[old].prev
	next := lru.elements[old].next
	lru.elements[prev].next = new
	lru.elements[next].prev = new

	prev = lru.elements[old].prevBucket
	next = lru.elements[old].nextBucket
	lru.elements[prev].nextBucket = new
	lru.elements[next].prevBucket = new

	lru.elements[new] = lru.elements[old]

	if lru.buckets[lru.elements[new].bucketPos] == old {
		lru.buckets[lru.elements[new].bucketPos] = new
	}
}

// insert stores the k/v at pos.
// It updates the head to point to this position.
func (lru *LRU[K, V]) insert(pos uint32, key K, value V, lifetime time.Duration) {
	lru.elements[pos].key = key
	lru.elements[pos].value = value
	lru.elements[pos].expire = expire(lifetime)

	if lru.len == 0 {
		lru.elements[pos].prev = pos
		lru.elements[pos].next = pos
		lru.head = pos
	} else if pos != lru.head {
		lru.setHead(pos)
	}
	lru.len++
	lru.inserts++
}

func now() int64 {
	return time.Now().UnixMilli()
}

func expire(lifetime time.Duration) int64 {
	if lifetime == 0 {
		return 0
	}
	return now() + lifetime.Milliseconds()
}

// clearKeyAndValue clears stale data to avoid memory leaks
func (lru *LRU[K, V]) clearKeyAndValue(pos uint32) {
	lru.elements[pos].key = lru.emptyKey
	lru.elements[pos].value = lru.emptyValue
}

func (lru *LRU[K, V]) findKey(key K) (uint32, bool) {
	_, startPos := lru.keyToPos(key)
	if startPos == emptyBucket {
		return emptyBucket, false
	}

	pos := startPos
	for {
		if key == lru.elements[pos].key {
			if lru.elements[pos].expire != 0 && lru.elements[pos].expire <= now() {
				lru.clearKeyAndValue(pos)
				return emptyBucket, false
			}
			return pos, true
		}

		pos = lru.elements[pos].nextBucket
		if pos == startPos {
			// Key not found
			return emptyBucket, false
		}
	}
}

// Len returns the number of elements stored in the cache.
func (lru *LRU[K, V]) Len() int {
	return int(lru.len)
}

// AddWithExpire adds a key:value to the cache with a lifetime.
// Returns true, true if key was updated and eviction occurred.
func (lru *LRU[K, V]) AddWithExpire(key K, value V, lifetime time.Duration) (evicted bool) {
	bucketPos, startPos := lru.keyToPos(key)
	if startPos == emptyBucket {
		pos := lru.len

		if pos == lru.cap {
			// Capacity reached, evict the oldest entry and
			// store the new entry at evicted position.
			pos = lru.elements[lru.head].next
			lru.evict(pos)
			evicted = true
		}

		// insert new (first) entry into the bucket
		lru.buckets[bucketPos] = pos
		lru.elements[pos].bucketPos = bucketPos

		lru.elements[pos].nextBucket = pos
		lru.elements[pos].prevBucket = pos
		lru.insert(pos, key, value, lifetime)
		return
	}

	// Walk through the bucket list to see whether key already exists.
	pos := startPos
	for {
		if lru.elements[pos].key == key {
			// Key exists, replace the value and update element to be the head element.
			lru.elements[pos].value = value
			lru.elements[pos].expire = expire(lifetime)

			if pos != lru.head {
				lru.unlinkElement(pos)
				lru.setHead(pos)
			}
			// count as insert, even if it's just an update
			lru.inserts++
			return false
		}

		pos = lru.elements[pos].nextBucket
		if pos == startPos {
			// Key not found
			break
		}
	}

	pos = lru.len
	if pos == lru.cap {
		// Capacity reached, evict the oldest entry and
		// store the new entry at evicted position.
		pos = lru.elements[lru.head].next
		lru.evict(pos)
		evicted = true
	}

	// insert new entry into the existing bucket before startPos
	lru.buckets[bucketPos] = pos
	lru.elements[pos].bucketPos = bucketPos
	lru.elements[pos].nextBucket = startPos
	lru.elements[pos].prevBucket = lru.elements[startPos].prevBucket
	lru.elements[lru.elements[startPos].prevBucket].nextBucket = pos
	lru.elements[startPos].prevBucket = pos
	lru.insert(pos, key, value, lifetime)

	if lru.elements[pos].prevBucket != pos {
		// The bucket now contains more than 1 element.
		// That means we have a collision.
		lru.collisions++
	}
	return
}

// Add adds a key:value to the cache.
// Returns true, true if key was updated and eviction occurred.
func (lru *LRU[K, V]) Add(key K, value V) (evicted bool) {
	return lru.AddWithExpire(key, value, lru.lifetime)
}

// Get looks up a key's value from the cache, setting it as the most
// recently used item.
func (lru *LRU[K, V]) Get(key K) (value V, ok bool) {
	if pos, ok := lru.findKey(key); ok {
		if pos != lru.head {
			lru.unlinkElement(pos)
			lru.setHead(pos)
		}
		return lru.elements[pos].value, ok
	}

	return
}

// Peek looks up a key's value from the cache, without changing its recent-ness.
func (lru *LRU[K, V]) Peek(key K) (value V, ok bool) {
	if pos, ok := lru.findKey(key); ok {
		return lru.elements[pos].value, ok
	}

	return
}

// Contains checks for the existence of a key, without changing its recent-ness.
func (lru *LRU[K, V]) Contains(key K) (ok bool) {
	_, ok = lru.Peek(key)
	return
}

// Remove removes the key from the cache.
// The return value indicates whether the key existed or not.
func (lru *LRU[K, V]) Remove(key K) (removed bool) {
	if pos, ok := lru.findKey(key); ok {
		// Key exists, update element to be the head element.
		lru.evict(pos)
		lru.move(pos, lru.len)
		lru.removals++

		// remove stale data to avoid memory leaks
		lru.clearKeyAndValue(lru.len)
		return ok
	}

	return
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

// Purge purges all data (key and value) from the LRU.
func (lru *LRU[K, V]) Purge() {
	for i := range lru.buckets {
		lru.buckets[i] = emptyBucket
	}

	for i := range lru.elements {
		lru.elements[i].key = lru.emptyKey
		lru.elements[i].value = lru.emptyValue
	}

	lru.len = 0
	lru.collisions = 0
	lru.inserts = 0
	lru.evictions = 0
	lru.removals = 0
}

// just used for debugging
func (lru *LRU[K, V]) dump() {
	fmt.Printf("head %d len %d cap %d size %d mask 0x%X\n",
		lru.head, lru.len, lru.cap, lru.size, lru.mask)

	for i := range lru.buckets {
		if lru.buckets[i] == emptyBucket {
			continue
		}
		fmt.Printf("  bucket[%d] -> %d\n", i, lru.buckets[i])
		pos := lru.buckets[i]
		for {
			e := &lru.elements[pos]
			fmt.Printf("    pos %d bucketPos %d prevBucket %d nextBucket %d prev %d next %d k %v v %v\n",
				pos, e.bucketPos, e.prevBucket, e.nextBucket, e.prev, e.next, e.key, e.value)
			pos = e.nextBucket
			if pos == lru.buckets[i] {
				break
			}
		}
	}
}

func (lru *LRU[K, V]) PrintStats() {
	fmt.Printf("Inserts: %d Collisions: %d (%.2f%%) Evictions: %d Removals: %d\n",
		lru.inserts, lru.collisions,
		float64(lru.collisions)/float64(lru.inserts),
		lru.evictions, lru.removals)
}
