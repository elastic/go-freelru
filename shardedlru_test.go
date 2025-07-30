// nolint:dupl // code duplication is ok for testing purposes
package freelru

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// TestShardedRaceCondition tests that the sharded LRU is safe to use concurrently.
// Test with 'go test . -race'.
func TestShardedRaceCondition(t *testing.T) {
	const CAP = 4

	lru, err := NewSharded[uint64, int](CAP, hashUint64)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	wg := sync.WaitGroup{}

	call := func(fn func()) {
		wg.Add(1)
		go func() {
			fn()
			wg.Done()
		}()
	}

	call(func() { lru.SetLifetime(1) })
	call(func() { lru.SetOnEvict(nil) })
	call(func() { _ = lru.Len() })
	call(func() { _ = lru.AddWithLifetime(1, 1, 0) })
	call(func() { _ = lru.Add(1, 1) })
	call(func() { _, _ = lru.Get(1) })
	call(func() { _, _ = lru.GetAndRefresh(1, 0) })
	call(func() { _, _ = lru.Peek(1) })
	call(func() { _ = lru.Contains(1) })
	call(func() { _ = lru.Remove(1) })
	call(func() { _, _, _ = lru.RemoveOldest() })
	call(func() { _ = lru.Keys() })
	call(func() { lru.Purge() })
	call(func() { lru.PurgeExpired() })
	call(func() { lru.Metrics() })
	call(func() { _ = lru.ResetMetrics() })
	call(func() { lru.dump() })
	call(func() { lru.PrintStats() })

	wg.Wait()
}

func TestShardedLRUMetrics(t *testing.T) {
	cache, _ := NewSharded[uint64, uint64](1, hashUint64)
	testMetrics(t, cache)
}

func TestStressWithLifetime(t *testing.T) {
	const CAP = 1024

	lru, err := NewSharded[string, int](CAP, hashString)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	lru.SetLifetime(time.Millisecond * 10)

	const NTHREADS = 10
	const RUNS = 1000

	wg := sync.WaitGroup{}

	for i := 0; i < NTHREADS; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < RUNS; i++ {
				//nolint:gosec // weak random is okay for testing
				lru.Add(fmt.Sprintf("key-%d", rand.Int()%1000), rand.Int())
				time.Sleep(time.Millisecond * 1)
			}
			wg.Done()
		}()
	}

	for i := 0; i < NTHREADS; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < RUNS; i++ {
				//nolint:gosec // weak random is okay for testing
				_, _ = lru.Get(fmt.Sprintf("key-%d", rand.Int()%1000))
				time.Sleep(time.Millisecond * 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()
}

// hashString is a simple but sufficient string hash function.
func hashString(s string) uint32 {
	var h uint32
	for i := 0; i < len(s); i++ {
		h = h*31 + uint32(s[i])
	}
	return h
}

func TestShardedLRU_RemoveOldest(t *testing.T) {
	evictCounter := uint64(0)
	testCacheRemoveOldest(t, makeShardedLRU(t, 2, &evictCounter), &evictCounter)
}