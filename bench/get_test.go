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

package benchmarks

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
	"github.com/elastic/go-freelru"
	"github.com/hashicorp/golang-lru/v2/simplelru"
)

func BenchmarkFreeLRUGet(b *testing.B) {
	lru, err := freelru.New[int, int](CAP, hashIntAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		// nolint:gosec
		val := int(rand.Int63())
		lru.Add(i, val)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = lru.Get(i)
	}
}

func BenchmarkSimpleLRUGet(b *testing.B) {
	lru, err := simplelru.NewLRU[int, int](CAP, nil)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		// nolint:gosec
		val := int(rand.Int63())
		lru.Add(i, val)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = lru.Get(i)
	}
}

func BenchmarkFreeCacheGet(b *testing.B) {
	lru := freecache.NewCache(CAP)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < CAP; i++ {
		// nolint:gosec
		val := int(rand.Int63())
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], uint64(val))
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(i))
		_ = lru.Set(bk[:], bv[:], 60)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(i))
		bv, err := lru.Get(bk[:])
		if err == nil {
			val = binary.BigEndian.Uint64(bv)
			_ = val
		}
	}
}

func BenchmarkRistrettoGet(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		// nolint:gosec
		val := int(rand.Int63())
		cache.Set(i, val, 1)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(i)
	}
}

func BenchmarkMapGet(b *testing.B) {
	cache := make(map[int]int, CAP)

	for i := 0; i < CAP; i++ {
		// nolint:gosec
		val := int(rand.Int63())
		cache[i] = val
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cache[i]
	}
}
