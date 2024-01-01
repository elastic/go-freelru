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
	"fmt"
	"math/rand"
	"runtime"
	"testing"

	"github.com/coocood/freecache"

	"github.com/elastic/go-freelru"
)

func runParallelSyncedFreeLRUAdd[K comparable, V any](b *testing.B) {
	lru, err := freelru.NewSynced[K, V](CAP, getHashAESENC[K]())
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	var val V
	keys := getParallelKeys[K]()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(len(keys)); pb.Next(); i++ {
			if i >= len(keys) {
				i = 0
			}
			lru.Add(keys[i], val)
		}
	})
}

func runParallelShardedFreeLRUAdd[K comparable, V any](b *testing.B) {
	lru, err := freelru.NewSharded[K, V](CAP, getHashAESENC[K]())
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	var val V
	keys := getParallelKeys[K]()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(len(keys)); pb.Next(); i++ {
			if i >= len(keys) {
				i = 0
			}
			lru.Add(keys[i], val)
		}
	})
}

func BenchmarkParallelSyncedFreeLRUAdd_int_int(b *testing.B) {
	runParallelSyncedFreeLRUAdd[int, int](b)
}

func BenchmarkParallelSyncedFreeLRUAdd_int_int128(b *testing.B) {
	runParallelSyncedFreeLRUAdd[int, int128](b)
}

func BenchmarkParallelShardedFreeLRUAdd_int_int(b *testing.B) {
	runParallelShardedFreeLRUAdd[int, int](b)
}

func BenchmarkParallelShardedFreeLRUAdd_int_int128(b *testing.B) {
	runParallelShardedFreeLRUAdd[int, int128](b)
}

func BenchmarkParallelFreeCacheAdd_int_int(b *testing.B) {
	lru := freecache.NewCache(CAP)

	var val int
	keys := parallelIntKeys

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(len(keys)); pb.Next(); i++ {
			if i >= len(keys) {
				i = 0
			}
			bv := [8]byte{}
			binary.BigEndian.PutUint64(bv[:], uint64(val))
			bk := [8]byte{}
			binary.BigEndian.PutUint64(bk[:], uint64(keys[i]))
			_ = lru.Set(bk[:], bv[:], 60)
		}
	})
}

func BenchmarkParallelFreeCacheAdd_int_int128(b *testing.B) {
	lru := freecache.NewCache(CAP)

	var val int128
	keys := parallelIntKeys

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := rand.Intn(len(keys)); pb.Next(); i++ {
			if i >= len(keys) {
				i = 0
			}
			bv := [16]byte{}
			binary.BigEndian.PutUint64(bv[:], val.hi)
			binary.BigEndian.PutUint64(bv[8:], val.lo)
			bk := [8]byte{}
			binary.BigEndian.PutUint64(bk[:], uint64(keys[i]))
			_ = lru.Set(bk[:], bv[:], 60)
		}
	})
}

var (
	parallelStringKeys []string
	parallelIntKeys    []int
)

func init() {
	parallelStringKeys = makeParallelKeys[string](func(threadID, n int) string {
		return fmt.Sprintf("key-%04d-%10d", threadID, rand.Uint64()) //nolint:gosec
	})

	parallelIntKeys = makeParallelKeys[int](func(threadID, n int) int {
		return int(rand.Uint64()) //nolint:gosec
	})
}

func makeParallelKeys[K comparable](makeKey func(threadID, n int) K) []K {
	nThreads := runtime.GOMAXPROCS(0)
	keys := make([]K, nThreads*CAP)

	for threadID := 0; threadID < nThreads; threadID++ {
		for i := 0; i < CAP; i++ {
			keys[threadID*CAP+i] = makeKey(threadID, i)
		}
	}

	return keys
}

func getParallelKeys[K comparable]() []K {
	var k K
	switch any(k).(type) {
	case string:
		return any(parallelStringKeys).([]K)
	case int:
		return any(parallelIntKeys).([]K)
	}
	return nil
}
