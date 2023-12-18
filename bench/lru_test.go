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
	"strconv"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
	"github.com/elastic/go-freelru"
	"github.com/hashicorp/golang-lru/v2/simplelru"
	"github.com/zeebo/xxh3"
)

const CAP = 8192

func BenchmarkHashInt_FNV1A(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashIntFNV1A(i)
	}
}

func BenchmarkHashInt_AESENC(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashIntAESENC(i)
	}
}

func BenchmarkHashInt_XXHASH(b *testing.B) {
	bv := [8]byte{}
	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint64(bv[:], uint64(i))
		_ = xxhash.Sum64(bv[:])
	}
}

func BenchmarkHashInt_XXH3HASH(b *testing.B) {
	bv := [8]byte{}
	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint64(bv[:], uint64(i))
		_ = xxh3.Hash(bv[:])
	}
}

var testString = "test123 dlfksdlföls sdfsdlskdg sgksgjdgs gdkfggk"

func BenchmarkHashString_FNV1A(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashStringFNV1A(testString)
	}
}

func BenchmarkHashString_AESENC(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashStringAESENC(testString)
	}
}

func BenchmarkHashString_XXHASH(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashStringXXHASH(testString)
	}
}

func BenchmarkHashString_XXH3HASH(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashStringXXH3HASH(testString)
	}
}

func runFreeLRUAddInt[V any](b *testing.B) {
	lru, err := freelru.New[int, V](CAP, hashIntAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	keys := make([]int, 0, b.N)
	for i := 0; i < b.N; i++ {
		keys = append(keys, rand.Int())
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		lru.Add(keys[i], val)
	}
}

func runFreeLRUAddIntAscending[V any](b *testing.B) {
	lru, err := freelru.New[int, V](CAP, hashIntAESENC)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		lru.Add(i, val)
	}
}

func BenchmarkFreeLRUAdd_int_int(b *testing.B) {
	runFreeLRUAddInt[int](b)
}

func BenchmarkFreeLRUAdd_int_int128(b *testing.B) {
	runFreeLRUAddInt[int128](b)
}

func BenchmarkFreeLRUAdd_int_int_Ascending(b *testing.B) {
	runFreeLRUAddIntAscending[int](b)
}

func BenchmarkFreeLRUAdd_int_int128_Ascending(b *testing.B) {
	runFreeLRUAddIntAscending[int128](b)
}

func BenchmarkFreeLRUAdd_uint32_uint64(b *testing.B) {
	lru, err := freelru.New[uint32, uint64](CAP, hashUInt32)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(uint32(i), val)
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

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lru.Add(i, testString)
	}
}

func runSimpleLRUAddInt[V any](b *testing.B) {
	lru, err := simplelru.NewLRU[int, V](CAP, nil)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		lru.Add(i, val)
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

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		lru.Add(uint32(i), val)
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

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lru.Add(i, testString)
	}
}

func BenchmarkFreeCacheAdd_int_int(b *testing.B) {
	lru := freecache.NewCache(CAP)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(i))
		_ = lru.Set(bk[:], bv[:], 60)
	}
}

func BenchmarkFreeCacheAdd_int_int128(b *testing.B) {
	lru := freecache.NewCache(CAP)

	b.ReportAllocs()
	b.ResetTimer()

	var val int128
	for i := 0; i < b.N; i++ {
		bv := [16]byte{}
		binary.BigEndian.PutUint64(bv[:], val.hi)
		binary.BigEndian.PutUint64(bv[8:], val.lo)
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(i))
		_ = lru.Set(bk[:], bv[:], 60)
	}
}

func BenchmarkFreeCacheAdd_uint32_uint64(b *testing.B) {
	lru := freecache.NewCache(CAP)

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		bv := [8]byte{}
		binary.BigEndian.PutUint64(bv[:], val)
		bk := [8]byte{}
		binary.BigEndian.PutUint32(bk[:], uint32(i))
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

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bk := [8]byte{}
		binary.BigEndian.PutUint64(bk[:], uint64(i))
		_ = lru.Set(bk[:], []byte(testString), 60)
	}
}

func runRistrettoLRUAddInt[V any](b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		cache.Set(i, val, 1)
	}
}

func BenchmarkRistrettoAdd_int_int(b *testing.B) {
	runRistrettoLRUAddInt[int](b)
}

func BenchmarkRistrettoAdd_int128_int(b *testing.B) {
	runRistrettoLRUAddInt[int128](b)
}

func BenchmarkRistrettoAdd_uint32_uint64(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val uint64
	for i := 0; i < b.N; i++ {
		cache.Set(uint32(i), val, 1)
	}
}

func BenchmarkRistrettoAdd_string_uint64(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
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
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: CAP * 10, // number of keys to track frequency of.
		MaxCost:     CAP,      // maximum cost of cache.
		BufferItems: 64,       // number of keys per Get buffer.
	})
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	for i := 0; i < b.N; i++ {
		cache.Set(i, testString, 1)
	}
}

func runMapAddInt[V any](b *testing.B) {
	cache := make(map[int]V, b.N) // b.N to avoid reallocations

	b.ReportAllocs()
	b.ResetTimer()

	var val V
	for i := 0; i < b.N; i++ {
		cache[i] = val
	}
}

func BenchmarkMapAdd_int_int(b *testing.B) {
	runMapAddInt[int](b)
}

func BenchmarkMapAdd_int_int128(b *testing.B) {
	runMapAddInt[int128](b)
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

// This test is for memory comparison with FreeLRU and Go map.
//
// GOGC=off go test -memprofile=mem.out -test.memprofilerate=1 -count 1 -run SimpleLRUAdd
// go tool pprof mem.out
// (then check the top10)
func TestSimpleLRUAdd(t *testing.T) {
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
