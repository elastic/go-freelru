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
	"context"
	"encoding/binary"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/hashicorp/golang-lru/v2/simplelru"

	"github.com/elastic/go-freelru"
)

const CAP = 8192

func runFreeLRUAddInt[V any](b *testing.B) {
	lru, err := freelru.New[int, V](CAP, hashIntAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkFreeLRUAdd_int_int(b *testing.B) {
	runFreeLRUAddInt[int](b)
}

func BenchmarkFreeLRUAdd_int_int128(b *testing.B) {
	runFreeLRUAddInt[int128](b)
}

func BenchmarkFreeLRUAdd_uint32_uint64(b *testing.B) {
	lru, err := freelru.New[uint32, uint64](CAP, hashUInt32)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeUInt32s(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkFreeLRUAdd_string_uint64(b *testing.B) {
	lru, err := freelru.New[string, uint64](CAP, hashStringAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeStrings(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkFreeLRUAdd_int_string(b *testing.B) {
	lru, err := freelru.New[int, string](CAP, hashIntFNV1A)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], testString)
	}
}

func runSyncedFreeLRUAddInt[V any](b *testing.B) {
	lru, err := freelru.NewSynced[int, V](CAP, hashIntAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkSyncedFreeLRUAdd_int_int(b *testing.B) {
	runSyncedFreeLRUAddInt[int](b)
}

func BenchmarkSyncedFreeLRUAdd_int_int128(b *testing.B) {
	runSyncedFreeLRUAddInt[int128](b)
}

func BenchmarkSyncedFreeLRUAdd_uint32_uint64(b *testing.B) {
	lru, err := freelru.NewSynced[uint32, uint64](CAP, hashUInt32)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeUInt32s(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkSyncedFreeLRUAdd_string_uint64(b *testing.B) {
	lru, err := freelru.NewSynced[string, uint64](CAP, hashStringAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeStrings(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkSyncedFreeLRUAdd_int_string(b *testing.B) {
	lru, err := freelru.NewSynced[int, string](CAP, hashIntFNV1A)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], testString)
	}
}

func runShardedFreeLRUAddInt[V any](b *testing.B) {
	lru, err := freelru.NewSharded[int, V](CAP, hashIntAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkShardedFreeLRUAdd_int_int(b *testing.B) {
	runShardedFreeLRUAddInt[int](b)
}

func BenchmarkShardedFreeLRUAdd_int_int128(b *testing.B) {
	runShardedFreeLRUAddInt[int128](b)
}

func BenchmarkShardedFreeLRUAdd_uint32_uint64(b *testing.B) {
	lru, err := freelru.NewSharded[uint32, uint64](CAP, hashUInt32)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeUInt32s(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkShardedFreeLRUAdd_string_uint64(b *testing.B) {
	lru, err := freelru.NewSharded[string, uint64](CAP, hashStringAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeStrings(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkShardedFreeLRUAdd_int_string(b *testing.B) {
	lru, err := freelru.NewSharded[int, string](CAP, hashIntFNV1A)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], testString)
	}
}

func runSimpleLRUAddInt[V any](b *testing.B) {
	lru, err := simplelru.NewLRU[int, V](CAP, nil)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkSimpleLRUAdd_int_int(b *testing.B) {
	runSimpleLRUAddInt[int](b)
}

func BenchmarkSimpleLRUAdd_int_int128(b *testing.B) {
	runSimpleLRUAddInt[int128](b)
}

func BenchmarkSimpleLRUAdd_uint32_uint64(b *testing.B) {
	lru, err := simplelru.NewLRU[uint32, uint64](CAP, nil)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeUInt32s(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkSimpleLRUAdd_string_uint64(b *testing.B) {
	lru, err := simplelru.NewLRU[string, uint64](CAP, nil)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeStrings(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func BenchmarkSimpleLRUAdd_int_string(b *testing.B) {
	lru, err := simplelru.NewLRU[int, string](CAP, nil)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], testString)
	}
}

func BenchmarkFreeCacheAdd_int_int(b *testing.B) {
	lru := freecache.NewCache(CAP)

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(keys[i]))
		_ = lru.Set(bk[:], bv[:], 60)
	}
}

func BenchmarkFreeCacheAdd_int_int128(b *testing.B) {
	lru := freecache.NewCache(CAP)

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val int128
	for i := 0; i < b.N; i++ {
		bv := [16]byte{}
		binary.BigEndian.PutUint64(bv[:], val.hi)
		binary.BigEndian.PutUint64(bv[8:], val.lo)
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(keys[i]))
		_ = lru.Set(bk[:], bv[:], 60)
	}
}

func BenchmarkFreeCacheAdd_uint32_uint64(b *testing.B) {
	lru := freecache.NewCache(CAP)

	keys := makeUInt32s(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		bk := [8]byte{}
		binary.BigEndian.PutUint32(bk[:], keys[i])
		_ = lru.Set(bk[:], bv[:], 60)
	}
}

func BenchmarkFreeCacheAdd_string_uint64(b *testing.B) {
	lru := freecache.NewCache(CAP)

	keys := makeStrings(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		_ = lru.Set([]byte(keys[i]), bv[:], 60)
	}
}

func BenchmarkFreeCacheAdd_int_string(b *testing.B) {
	lru := freecache.NewCache(CAP)

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(keys[i]))
		_ = lru.Set(bk[:], []byte(testString), 60)
	}
}

func runRistrettoLRUAddInt[V any](b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config[int, V]{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		cache.Set(keys[i], val, 1)
	}
}

func BenchmarkRistrettoAdd_int_int(b *testing.B) {
	runRistrettoLRUAddInt[int](b)
}

func BenchmarkRistrettoAdd_int_int128(b *testing.B) {
	runRistrettoLRUAddInt[int128](b)
}

func BenchmarkRistrettoAdd_uint32_uint64(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config[uint32, uint64]{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeUInt32s(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		cache.Set(keys[i], val, 1)
	}
}

func BenchmarkRistrettoAdd_string_uint64(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, uint64]{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeStrings(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		cache.Set(keys[i], val, 1)
	}
}

func BenchmarkRistrettoAdd_int_string(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config[int, string]{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(keys[i], testString, 1)
	}
}

func newBigCache() *bigcache.BigCache {
	// These values have been taken from
	//   https://github.com/allegro/bigcache-bench/blob/master/caches_bench_test.go .
	lru, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             256,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 10000,
		MaxEntrySize:       256,
		Verbose:            false,
	})
	return lru
}

func BenchmarkBigCacheAdd_int_int(b *testing.B) {
	lru := newBigCache()

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		// Converting key to string and value to []byte counts into the benchmark because
		// the conversion is a required extra step unique to BigCache.
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(keys[i]))
		_ = lru.Set(string(bk[:]), bv[:])
	}
}

func BenchmarkBigCacheAdd_int_int128(b *testing.B) {
	lru := newBigCache()

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val int128
	for i := 0; i < b.N; i++ {
		// Converting key to string and value to []byte counts into the benchmark because
		// the conversion is a required extra step unique to BigCache.
		bv := [16]byte{}
		binary.BigEndian.PutUint64(bv[:], val.hi)
		binary.BigEndian.PutUint64(bv[8:], val.lo)
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(keys[i]))
		_ = lru.Set(string(bk[:]), bv[:])
	}
}

func BenchmarkBigCacheAdd_uint32_uint64(b *testing.B) {
	lru := newBigCache()

	keys := makeUInt32s(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		bk := [8]byte{}
		binary.BigEndian.PutUint32(bk[:], keys[i])
		_ = lru.Set(string(bk[:]), bv[:])
	}
}

func BenchmarkBigCacheAdd_string_uint64(b *testing.B) {
	lru := newBigCache()

	keys := makeStrings(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		_ = lru.Set(keys[i], bv[:])
	}
}

func BenchmarkBigCacheAdd_int_string(b *testing.B) {
	lru := newBigCache()

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(keys[i]))
		_ = lru.Set(string(bk[:]), []byte(testString))
	}
}

func runMapAddInt[V any](b *testing.B) {
	cache := make(map[int]V, b.N) // b.N to avoid reallocations

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		cache[keys[i]] = val
	}
}

func BenchmarkMapAdd_int_int(b *testing.B) {
	runMapAddInt[int](b)
}

func BenchmarkMapAdd_int_int128(b *testing.B) {
	runMapAddInt[int128](b)
}

func BenchmarkMapAdd_uint32_uint64(b *testing.B) {
	cache := make(map[uint32]uint64, b.N) // b.N to avoid reallocations

	keys := makeUInt32s(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		cache[keys[i]] = val
	}
}

func BenchmarkMapAdd_string_uint64(b *testing.B) {
	cache := make(map[string]uint64, b.N) // b.N to avoid reallocations

	keys := makeStrings(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		cache[keys[i]] = val
	}
}

func BenchmarkMapAdd_int_string(b *testing.B) {
	cache := make(map[int]string, b.N) // b.N to avoid reallocations

	keys := makeInts(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache[keys[i]] = testString
	}
}

// This test is for memory comparison with FreeLRU and Go map.
//
// GOGC=off go test -memprofile=mem.out -test.memprofilerate=1 -count 1 -run SimpleLRUAdd
// go tool pprof mem.out
// (then check the top10)
func TestSimpleLRUAdd(_ *testing.T) {
	cache, _ := simplelru.NewLRU[uint64, int](CAP, nil)

	var val int
	for i := uint64(0); i < 1000; i++ {
		cache.Add(i, val)
	}
}

func makeStrings(n int) []string {
	s := make([]string, 0, n)
	for i := 0; i < n; i++ {
		s = append(s, "heyja"+strconv.Itoa(i))
	}
	return s
}

func makeInts(n int) []int {
	s := make([]int, 0, n)
	for i := 0; i < n; i++ {
		s = append(s, rand.Int()) //nolint:gosec
	}
	return s
}

func makeUInt32s(n int) []uint32 {
	s := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		s = append(s, uint32(rand.Int())) //nolint:gosec
	}
	return s
}
