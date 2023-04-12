# FreeLRU - A GC-less, fast and generic LRU hashmap library for Go

FreeLRU allows you to cache objects without introducing GC overhead.
It uses Go generics for simplicity, type-safety and performance over interface types.
It uses a fast exact LRU algorithm.
It performs better than other LRU implementations in the Go benchmarks provided.
The API is simple in order to ease migrations from other LRU implementations.

FreeLRU is for single-threaded use only.
For thread-safety, the locking of operations needs to be controlled by the caller.

The function to calculate hashes from the keys needs to be provided by the caller.

### Merging hashmap and ringbuffer

Most LRU implementations combine Go's `map` for the key/value lookup and their own implementation of
a circular doubly-linked list for keeping track of the recent-ness of objects.
This requires one additional heap allocation for the list element. A second downside is that the list
elements are not contiguous in memory, which causes more (expensive) CPU cache misses for accesses.

FreeLRU addresses both issues by merging hashmap and ringbuffer into a contiguous array of elements.
Each element contains key, value and two indices to keep the cached objects ordered by recent-ness.

### Avoiding GC overhead

The contiguous array of elements is allocated on cache creation time.
So there is only a single memory object instead of possibly millions that the GC needs to iterate during
a garbage collection phase.
The GC overhead can be quite large in comparison with the overall CPU usage of an application.
Especially long-running and low-CPU applications with lots of cached objects suffer from the GC overhead.

### Type safety by using generics

Using generics allows type-checking at compile time, so type conversions are not needed at runtime.
The interface type or `any` requires type conversions at runtime, which may fail.

### Reducing memory allocations by using generics

The interface types (aka `any`) is a pointer type and thus require a heap allocation when being stored.
This is true even if you just need an integer to integer lookup or translation.

With generics, the two allocations for key and value can be avoided: as long as the key and value types do not contain
pointer types, no allocations will take place when adding such objects to the cache.

### Overcommitting of hashtable memory

Each hashtable implementation tries to avoid hash collisions because collisions are expensive.
FreeLRU allows allocating more elements than the maximum number of elements stored.
This value is configurable and can be increased to reduce the likeliness of collisions.
The performance of the LRU operations will generally become faster by doing so.
Setting the size of LRU to a value of 2^N is recognized to replace slow divisions by fast bitwise AND operations.

## Benchmarks

Below we compare FreeLRU with SimpleLRU, FreeCache and Go map.
The comparison with FreeCache is just for reference - it is thread-safe and comes with a mutex/locking overhead.
The comparison with Go map is also just for reference - Go maps don't implement LRU functionality and thus should
be significantly faster than FreeLRU. It turns out, the opposite is the case.

The numbers are from my laptop (Intel(R) Core(TM) i7-12800H @ 2800 MHz).

The key and value types are part of the benchmark name, e.g. `int_int` means key and value are of type `int`.
`int128` is a struct type made of two `uint64` fields.

To run the benchmarks
```
go test -count 1 -bench . -run XXX
```

### Adding objects

FreeLRU is ~3.5x faster than SimpleLRU, no surprise.
But it is also significantly faster than Go maps, which is a bit of a surprise.

This is with 0% memory overcommitment (default) and a capacity of 8192.
 ```
BenchmarkFreeLRUAdd_int_int-20                  43097347                27.41 ns/op            0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_int_int128-20               42129165                28.38 ns/op            0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_uint32_uint64-20            98322132                11.74 ns/op            0 B/op          0 allocs/op (*)
BenchmarkFreeLRUAdd_string_uint64-20            39122446                31.12 ns/op            0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_int_string-20               81920673                14.00 ns/op            0 B/op          0 allocs/op (*)
BenchmarkSimpleLRUAdd_int_int-20                12253708                93.85 ns/op           48 B/op          1 allocs/op
BenchmarkSimpleLRUAdd_int_int128-20             12095150                94.26 ns/op           48 B/op          1 allocs/op
BenchmarkSimpleLRUAdd_uint32_uint64-20          12367568                92.60 ns/op           48 B/op          1 allocs/op
BenchmarkSimpleLRUAdd_string_uint64-20          10395525               119.0 ns/op            49 B/op          1 allocs/op
BenchmarkSimpleLRUAdd_int_string-20             12373900                94.40 ns/op           48 B/op          1 allocs/op
BenchmarkFreeCacheAdd_int_int-20                 9691870               122.9 ns/op             1 B/op          0 allocs/op
BenchmarkFreeCacheAdd_int_int128-20              9240273               125.6 ns/op             1 B/op          0 allocs/op
BenchmarkFreeCacheAdd_uint32_uint64-20           8140896               132.1 ns/op             1 B/op          0 allocs/op
BenchmarkFreeCacheAdd_string_uint64-20           8248917               137.9 ns/op             1 B/op          0 allocs/op
BenchmarkFreeCacheAdd_int_string-20              8079253               145.0 ns/op            64 B/op          1 allocs/op
BenchmarkRistrettoAdd_int_int-20                11102623               100.1 ns/op           109 B/op          2 allocs/op
BenchmarkRistrettoAdd_int128_int-20             10317686               113.5 ns/op           129 B/op          4 allocs/op
BenchmarkRistrettoAdd_uint32_uint64-20          12892147                94.28 ns/op          104 B/op          2 allocs/op
BenchmarkRistrettoAdd_string_uint64-20          11266416               105.8 ns/op           122 B/op          3 allocs/op
BenchmarkRistrettoAdd_int_string-20             10360814               107.4 ns/op           129 B/op          4 allocs/op
BenchmarkMapAdd_int_int-20                      35306983                46.29 ns/op            0 B/op          0 allocs/op
BenchmarkMapAdd_int_int128-20                   30986126                45.16 ns/op            0 B/op          0 allocs/op
BenchmarkMapAdd_string_uint64-20                28406497                49.35 ns/op            0 B/op          0 allocs/op
```
(*)
There is an interesting affect when using increasing number (0..N) as keys in combination with FNV1a().
The number of collisions is strongly reduced here, thus the high performance.
Exchanging the sequential numbers with random numbers results in roughly the same performance as the other results.

Just to give you an idea for 100% memory overcommitment:
Performance increased by ~20%.
```
BenchmarkFreeLRUAdd_int_int-20                  53473030                21.52 ns/op            0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_int_int128-20               52852280                22.10 ns/op            0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_uint32_uint64-20            100000000               10.15 ns/op            0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_string_uint64-20            49477594                24.55 ns/op            0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_int_string-20               85288306                12.10 ns/op            0 B/op          0 allocs/op
```

### Getting objects

This is with 0% memory overcommitment (default) and a capacity of 8192.
```
BenchmarkFreeLRUGet-20                          83158561                13.80 ns/op            0 B/op          0 allocs/op
BenchmarkSimpleLRUGet-20                        146248706                8.199 ns/op           0 B/op          0 allocs/op
BenchmarkFreeCacheGet-20                        58229779                19.56 ns/op            0 B/op          0 allocs/op
BenchmarkRistrettoGet-20                        31157457                35.37 ns/op           10 B/op          1 allocs/op
BenchmarkMapGet-20                              195464706                6.031 ns/op           0 B/op          0 allocs/op
```

## Example usage

```go
package main

import (
	"fmt"

	"github.com/cespare/xxhash/v2"

	"github.com/elastic/go-freelru"
)

// more hash function in https://github.com/elastic/go-freelru/blob/main/bench/hash.go
func hashStringXXHASH(s string) uint32 {
	return uint32(xxhash.Sum64String(s))
}

func main() {
	lru, err := freelru.New[string, uint64](8192, hashStringXXHASH)
	if err != nil {
		panic(err)
	}

	key := "go-freelru"
	val := uint64(999)
	lru.Add(key, val)

	if v, ok := lru.Get(key); ok {
		fmt.Printf("found %v=%v\n", key, v)
	}

	// Output:
	// found go-freelru=999
}
```

The function `hashInt(int) uint32` will be called to calculate a hash value of the key.
Please take a look into `bench/` directory to find examples of hash functions.
Here you will also find an amd64 version of the Go internal hash function, which uses AESENC features
of the CPU.

In case you already have a hash that you want to use as the key, you have to provide an "identity" function.
