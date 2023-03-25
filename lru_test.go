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
	"encoding/binary"
	"math/rand"
	"testing"
	"time"
)

const (
	// FNV-1a
	offset32 = uint32(2166136261)
	prime32  = uint32(16777619)

	// Init32 is what 32 bits hash values should be initialized with.
	Init32 = offset32
)

func hashUint64(i uint64) uint32 {
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], i)
	h := Init32
	h = (h ^ uint32(b[0])) * prime32
	h = (h ^ uint32(b[1])) * prime32
	h = (h ^ uint32(b[2])) * prime32
	h = (h ^ uint32(b[3])) * prime32
	h = (h ^ uint32(b[4])) * prime32
	h = (h ^ uint32(b[5])) * prime32
	h = (h ^ uint32(b[6])) * prime32
	h = (h ^ uint32(b[7])) * prime32
	return h
}

func makeLRUWithLifetime(t *testing.T, cap uint32, evictCounter *int,
	lifetime time.Duration) *LRU[uint64, uint64] {
	onEvict := func(k uint64, v uint64) {
		FatalIf(t, k+1 != v, "Evict value not matching (%v+1 != %v)", k, v)
		if evictCounter != nil {
			*evictCounter++
		}
	}

	lru, err := New[uint64, uint64](cap, hashUint64)
	FatalIf(t, err != nil, "Failed to create LRU: %v", err)

	lru.SetLifetime(lifetime)
	lru.SetOnEvict(onEvict)

	return lru
}

func makeLRU(t *testing.T, cap uint32, evictCounter *int) *LRU[uint64, uint64] {
	return makeLRUWithLifetime(t, cap, evictCounter, 0)
}

func TestLRU(t *testing.T) {
	const CAP = 32

	evictCounter := 0
	lru := makeLRU(t, CAP, &evictCounter)

	for i := uint64(0); i < CAP*2; i++ {
		lru.Add(i, i+1)
	}
	FatalIf(t, lru.Len() != CAP, "Unexpected number of entries: %v (!= %d)", lru.Len(), CAP)
	FatalIf(t, evictCounter != CAP, "Unexpected number of evictions: %v (!= %d)", evictCounter, CAP)

	keys := lru.Keys()
	for i, k := range keys {
		if v, ok := lru.Get(k); !ok || v != k+1 || v != uint64(i+CAP+1) {
			t.Fatalf("Mismatch of key %v (ok=%v v=%v)\n%v", k, ok, v, keys)
		}
	}

	for i := uint64(0); i < CAP; i++ {
		if _, ok := lru.Get(i); ok {
			t.Fatalf("Missing eviction of %d", i)
		}
	}

	for i := uint64(CAP); i < CAP*2; i++ {
		if _, ok := lru.Get(i); !ok {
			t.Fatalf("Unexpected eviction of %d", i)
		}
	}

	FatalIf(t, lru.Remove(CAP*2), "Unexpected success removing %d", CAP*2)
	FatalIf(t, !lru.Remove(CAP*2-1), "Failed to remove most recent entry %d", CAP*2-1)
	FatalIf(t, !lru.Remove(CAP), "Failed to remove oldest entry %d", CAP)
	FatalIf(t, evictCounter != CAP+2, "Unexpected # of evictions: %d (!= %d)", evictCounter, CAP+2)
	FatalIf(t, lru.Len() != CAP-2, "Unexpected # of entries: %d (!= %d)", lru.Len(), CAP-2)
}

func TestLRU_Add(t *testing.T) {
	evictCounter := 0
	lru := makeLRU(t, 1, &evictCounter)

	FatalIf(t, lru.Add(1, 2) == true || evictCounter != 0, "Unexpected eviction")
	FatalIf(t, lru.Add(3, 4) == false || evictCounter != 1, "Missing eviction")
}

func TestLRU_Remove(t *testing.T) {
	evictCounter := 0
	lru := makeLRU(t, 2, &evictCounter)
	lru.Add(1, 2)
	lru.Add(3, 4)

	FatalIf(t, !lru.Remove(1), "Failed to remove most recent entry %d", 1)
	FatalIf(t, !lru.Remove(3), "Failed to remove most recent entry %d", 3)
	FatalIf(t, evictCounter != 2, "Unexpected # of evictions: %d (!= %d)", evictCounter, 2)
	FatalIf(t, lru.Len() != 0, "Unexpected # of entries: %d (!= %d)", lru.Len(), 0)
}

func TestLRU_AddWithExpire(t *testing.T) {
	// check for LRU default lifetime + element specific override
	lru := makeLRUWithLifetime(t, 2, nil, 100*time.Millisecond)
	lru.Add(1, 2)
	lru.AddWithExpire(3, 4, 200*time.Millisecond)
	_, ok := lru.Get(1)
	FatalIf(t, !ok, "Failed to get")
	time.Sleep(101 * time.Millisecond)
	_, ok = lru.Get(1)
	FatalIf(t, ok, "Expected expiration did not happen")
	_, ok = lru.Get(3)
	FatalIf(t, !ok, "Failed to get")
	time.Sleep(100 * time.Millisecond)
	_, ok = lru.Get(3)
	FatalIf(t, ok, "Expected expiration did not happen")

	// check for element specific lifetime
	lru = makeLRU(t, 1, nil)
	lru.AddWithExpire(1, 2, 100*time.Millisecond)
	_, ok = lru.Get(1)
	FatalIf(t, !ok, "Failed to get")
	time.Sleep(101 * time.Millisecond)
	_, ok = lru.Get(1)
	FatalIf(t, ok, "Expected expiration did not happen")
}

// Test that Go map and the LRU stay in sync when adding
// and randomly removing elements.
func TestLRUMatch(t *testing.T) {
	const CAP = 128

	backup := make(map[uint64]uint64, CAP)

	onEvict := func(k uint64, v uint64) {
		FatalIf(t, k != v, "Evict value not matching (%v != %v)", k, v)
		delete(backup, k)
	}

	lru, err := New[uint64, uint64](CAP, hashUint64)
	FatalIf(t, err != nil, "Failed to create LRU: %v", err)
	lru.SetOnEvict(onEvict)

	for i := uint64(0); i < 100000; i++ {
		lru.Add(i, i)
		backup[i] = i

		// ~33% chance to remove a random element
		r := i - uint64(rand.Int63()%(CAP*3)) // nolint:gosec
		lru.Remove(r)

		FatalIf(t, lru.Len() != len(backup), "Len does not match (%d vs %d)",
			lru.Len(), len(backup))

		keys := lru.Keys()
		FatalIf(t, len(keys) != len(backup), "Number of keys does not match (%d vs %d)",
			len(keys), len(backup))

		for _, key := range keys {
			backupVal, ok := backup[key]
			FatalIf(t, !ok, "Failed to find key %d in map", key)

			lruVal, ok := lru.Peek(key)
			FatalIf(t, !ok, "Failed to find key %d in LRU", key)

			FatalIf(t, backupVal != lruVal, "Unexpected mismatch of values: %#v %#v", backupVal, lruVal)
		}

		for k, v := range backup {
			lruVal, ok := lru.Peek(k)
			FatalIf(t, !ok, "Failed to find key %d in LRU (i=%d)", k, i)
			FatalIf(t, v != lruVal, "Unexpected mismatch of values: %#v %#v", v, lruVal)
		}
	}

	lru.PrintStats()
}

func FatalIf(t *testing.T, fail bool, fmt string, args ...any) {
	if fail {
		t.Logf(fmt, args...)
		panic(fail)
	}
}

// This following tests are for memory comparison.
// See also TestSimpleLRUAdd in the bench/ directory.

const count = 1000

// GOGC=off go test -memprofile=mem.out -test.memprofilerate=1 -count 1 -run MapAdd
// go tool pprof mem.out
// (then check the top10)
func TestMapAdd(t *testing.T) {
	cache := make(map[uint64]int, count) // b.N to avoid reallocations

	var val int
	for i := uint64(0); i < count; i++ {
		cache[i] = val
	}
}

// GOGC=off go test -memprofile=mem.out -test.memprofilerate=1 -count 1 -run FreeLRUAdd
// go tool pprof mem.out
// (then check the top10)
func TestFreeLRUAdd(t *testing.T) {
	cache, _ := New[uint64, int](count, hashUint64)

	var val int
	for i := uint64(0); i < count; i++ {
		cache.Add(i, val)
	}
}
