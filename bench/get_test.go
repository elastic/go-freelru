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
	"time"

	cloudflare "github.com/cloudflare/golibs/lrucache"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
	"github.com/hashicorp/golang-lru/v2/simplelru"
	phuslu "github.com/phuslu/lru"

	"github.com/elastic/go-freelru"
)

func BenchmarkFreeLRUGet(b *testing.B) {
	lru, err := freelru.New[int, int](CAP, hashIntAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		lru.Add(intKeys[i], intKeys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = lru.Get(intKey(i))
	}
}

func BenchmarkSimpleLRUGet(b *testing.B) {
	lru, err := simplelru.NewLRU[int, int](CAP, nil)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		lru.Add(intKeys[i], intKeys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = lru.Get(intKey(i))
	}
}

func BenchmarkSyncedGet(b *testing.B) {
	lru, err := freelru.NewSynced[int, int](CAP, hashIntFNV1A)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := make([]int, CAP)
	for i := 0; i < CAP; i++ {
		// nolint:gosec
		keys[i] = int(rand.Int63())
	}

	for i := 0; i < CAP; i++ {
		// nolint:gosec
		lru.Add(keys[i], i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Test with 50% hit rate.
	for i := 0; i < b.N; i++ {
		_, _ = lru.Get(intKey(i))
	}
}

func BenchmarkParallelSyncedGet(b *testing.B) {
	lru, err := freelru.NewSynced[int, int](CAP, hashIntFNV1A)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		_ = lru.Add(intKeys[i], intKeys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(CAP); pb.Next(); i++ {
			if i >= CAP {
				i = 0
			}
			_, _ = lru.Get(intKey(i))
		}
	})
}

func BenchmarkShardedGet(b *testing.B) {
	lru, err := freelru.NewShardedWithSize[int, int](1024, CAP, CAP*2, getHashAESENC[int]())
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		_ = lru.Add(intKeys[i], intKeys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = lru.Get(intKey(i))
	}
}

func BenchmarkParallelShardedGet(b *testing.B) {
	lru, err := freelru.NewShardedWithSize[int, int](1024, CAP, CAP*2, getHashAESENC[int]())
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		_ = lru.Add(intKeys[i], intKeys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(CAP); pb.Next(); i++ {
			if i >= CAP {
				i = 0
			}
			_, _ = lru.Get(intKey(i))
		}
	})
}

func BenchmarkFreeCacheGet(b *testing.B) {
	lru := freecache.NewCache(CAP)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < CAP; i++ {
		// nolint:gosec
		val := intKeys[i]
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], uint64(val))
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(val))
		_ = lru.Set(bk[:], bv[:], 60)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(intKey(i)))
		bv, err := lru.Get(bk[:])
		if err == nil {
			_ = binary.BigEndian.Uint64(bv)
		}
	}
}

func BenchmarkParallelFreeCacheGet(b *testing.B) {
	lru := freecache.NewCache(CAP)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < CAP; i++ {
		// nolint:gosec
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], uint64(intKeys[i]))
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(intKeys[i]))
		_ = lru.Set(bk[:], bv[:], 60)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(CAP); pb.Next(); i++ {
			if i >= CAP {
				i = 0
			}
			bk := [8]byte{}
			binary.BigEndian.PutUint64(bk[:], uint64(intKey(i)))
			bv, err := lru.Get(bk[:])
			if err == nil {
				_ = binary.BigEndian.Uint64(bv)
			}
		}
	})
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
		cache.Set(intKeys[i], intKeys[i], 1)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(intKey(i))
	}
}

func BenchmarkParallelRistrettoGet(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < CAP; i++ {
		cache.Set(intKeys[i], intKeys[i], 1)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(CAP); pb.Next(); i++ {
			if i >= CAP {
				i = 0
			}
			_, _ = cache.Get(intKey(i))
		}
	})
}

func BenchmarkBigCacheGet(b *testing.B) {
	cache := newBigCache()

	var val uint64
	for i := 0; i < CAP; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(intKeys[i]))
		_ = cache.Set(string(bk[:]), bv[:])
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Converting key to string counts into the benchmark because
		// the conversion is a required extra step unique to BigCache.
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(intKey(i)))
		bv, err := cache.Get(string(bk[:]))
		if err == nil {
			_ = binary.BigEndian.Uint64(bv)
		}
	}
}

func BenchmarkParallelBigCacheGet(b *testing.B) {
	cache := newBigCache()

	var val uint64
	for i := 0; i < CAP; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(intKeys[i]))
		_ = cache.Set(string(bk[:]), bv[:])
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(CAP); pb.Next(); i++ {
			if i >= CAP {
				i = 0
			}
			// Converting key to string counts into the benchmark because
			// the conversion is a required extra step unique to BigCache.
			bk := [8]byte{}
			binary.BigEndian.PutUint64(bk[:], uint64(intKey(i)))
			bv, err := cache.Get(string(bk[:]))
			if err == nil {
				_ = binary.BigEndian.Uint64(bv)
			}
		}
	})
}

func BenchmarkPhusluGet(b *testing.B) {
	cache := phuslu.New[int, int](CAP)

	for i := 0; i < CAP; i++ {
		_, _ = cache.Set(intKeys[i], intKeys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(intKey(i))
	}
}

func BenchmarkParallelPhusluGet(b *testing.B) {
	cache := phuslu.New[int, int](CAP)

	for i := 0; i < CAP; i++ {
		_, _ = cache.Set(intKeys[i], intKeys[i])
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(CAP); pb.Next(); i++ {
			if i >= CAP {
				i = 0
			}
			_, _ = cache.Get(intKey(i))
		}
	})
}

func BenchmarkCloudflareGet(b *testing.B) {
	// Only works with string as key.
	cache := cloudflare.NewMultiLRUCache(256, CAP/256)

	var expire time.Time
	for i := 0; i < CAP; i++ {
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(intKeys[i]))
		cache.Set(string(bk[:]), intKeys[i], expire)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Converting key to string counts into the benchmark because
		// the conversion is a required extra step needed for Cloudflare.
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(intKey(i)))
		_, _ = cache.Get(string(bk[:]))
	}
}

func BenchmarkParallelCloudflareGet(b *testing.B) {
	// Only works with string as key.
	cache := cloudflare.NewMultiLRUCache(256, CAP/256)

	var expire time.Time
	for i := 0; i < CAP; i++ {
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(intKeys[i]))
		cache.Set(string(bk[:]), intKeys[i], expire)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(CAP); pb.Next(); i++ {
			if i >= CAP {
				i = 0
			}
			// Converting key to string counts into the benchmark because
			// the conversion is a required extra step needed for Cloudflare.
			bk := [8]byte{}
			binary.BigEndian.PutUint64(bk[:], uint64(intKey(i)))
			_, _ = cache.Get(string(bk[:]))
		}
	})
}

func BenchmarkMapGet(b *testing.B) {
	cache := make(map[int]int, CAP)

	for i := 0; i < CAP; i++ {
		cache[i] = intKeys[i]
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cache[intKey(i)]
	}
}

// Use the same keys for all benchmarks.
var intKeys = make([]int, CAP)

func init() {
	for i := 0; i < len(intKeys); i++ {
		// nolint:gosec
		intKeys[i] = int(rand.Int63())
	}
}

func intKey(i int) int {
	// Test with 50% hit rate.
	if i&1 == 0 {
		return intKeys[i&(CAP-1)]
	}

	// For simplicity, we assume 'i' to not be in the cache.
	return i
}
