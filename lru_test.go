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
	"testing"
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

func makeLRU(t *testing.T, cap uint32, evictCounter *int) *LRU[uint64, uint64] {
	onEvicted := func(k uint64, v uint64) {
		if k+1 != v {
			t.Fatalf("Evict value not matching (%v+1 != %v)", k, v)
		}
		*evictCounter++
	}

	lru, err := New(cap, onEvicted, hashUint64)
	if err != nil {
		t.Fatalf("Failed to create LRU: %v", err)
	}

	return lru
}

func TestLRU(t *testing.T) {
	const CAP = 128

	evictCounter := 0
	lru := makeLRU(t, CAP, &evictCounter)

	for i := uint64(0); i < CAP*2; i++ {
		lru.Add(i, i+1)
	}
	if lru.Len() != CAP {
		t.Fatalf("Unexpected number of entries: %v (!= %d)", lru.Len(), CAP)
	}
	if evictCounter != CAP {
		t.Fatalf("Unexpected number of evictions: %v (!= %d)", evictCounter, CAP)
	}

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

	if lru.Remove(CAP * 2) {
		t.Fatalf("Unexpected success removing %d", CAP*2)
	}
	if !lru.Remove(CAP*2 - 1) {
		t.Fatalf("Failed to remove most recent entry %d", CAP*2-1)
	}
	if !lru.Remove(CAP) {
		t.Fatalf("Failed to remove oldest entry %d", CAP)
	}
	if evictCounter != CAP+2 {
		t.Fatalf("Unexpected number of evictions: %v (!= %d)", evictCounter, CAP+2)
	}
	if lru.Len() != CAP-2 {
		t.Fatalf("Unexpected number of entries: %v (!= %d)", lru.Len(), CAP-2)
	}
}

func TestLRU_Add(t *testing.T) {
	evictCounter := 0
	lru := makeLRU(t, 1, &evictCounter)

	if lru.Add(1, 2) == true || evictCounter != 0 {
		t.Errorf("Unexpected eviction")
	}
	if lru.Add(3, 4) == false || evictCounter != 1 {
		t.Errorf("Missing eviction")
	}
}

func TestLRU_Remove(t *testing.T) {
	evictCounter := 0
	lru := makeLRU(t, 2, &evictCounter)

	lru.Add(1, 2)
	lru.Add(3, 4)
	if !lru.Remove(1) {
		t.Fatalf("Failed to remove most recent entry %d", 1)
	}
	if !lru.Remove(3) {
		t.Fatalf("Failed to remove oldest entry %d", 3)
	}
	if evictCounter != 2 {
		t.Fatalf("Unexpected number of evictions: %v (!= %d)", evictCounter, 2)
	}
	if lru.Len() != 0 {
		t.Fatalf("Unexpected number of entries: %v (!= %d)", lru.Len(), 0)
	}
}
