# FreeLRU - A simple LRU cache library for Go (GC-less, generic)

FreeLRU allows you to cache objects without introducing GC overhead.
It uses a fast exact LRU algorithm that outperforms the widely used [golang-lru/simplelru](https://github.com/hashicorp/golang-lru/simplelru).
It uses Go generics for simplicity, type-safety and performance over interface types.
The API is simple to ease migrations from other LRU implementations.

FreeLRU is *not* concurrency safe (for single-threaded use only) as it aims to replace [golang-lru/simplelru](https://github.com/hashicorp/golang-lru/simplelru).

## Performance

The performance is gained by the combination of different techniques.

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

### Avoiding interface types by using generics

Interface types are pointer types and thus require a heap allocation for storing.
This is true even if you just need an integer to integer lookup or translation.

With generics, these two allocations can be avoided: as long as the key and value types do not contain
pointer types, no allocations will take place when adding such objects to the cache.

### Overcommitting of hashtable memory

Each hashtable implementation tries to avoid hash collisions because collisions are expensive.
FreeLRU does this by allocating more elements than technically are needed (25% more by default).
This value is configurable and can be increased to reduce the likelyness of collisions.

## Benchmarks

Below we compare FreeLRU with SimpleLRU, FreeCache and Go map.
The comparison with FreeCache is just for reference - it is thread-safe and thus it has a mutex overhead.
The comparison with Go map is also just for reference - Go maps don't implement LRU and thus should be much
faster than FreeLRU. I am surprised by the benchmark results.

The numbers are from my laptop (Intel(R) Core(TM) i5-8265U CPU @ 1.60GHz).

The key and value types are part of the benchmark name, e.g. `int_int` means both, key and value, are of type `int`.
`int128` is a struct type made of two `uint64` fields.

To run the benchamrks
```
go test -count 1 -bench . -run XXX
```

### Adding objects

FreeLRU is ~2.5x faster than SimpleLRU, no surprise.
But it is also ~27% and 34% faster than Go maps, which is a bit of a surprise.

This is with 25% memory overcommitment (default).
 ```
BenchmarkFreeLRUAdd_int_int-8           10254669               114.6 ns/op             0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_int_int128-8        10335757               115.6 ns/op             0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_int_string-8        10235534               116.9 ns/op             0 B/op          0 allocs/op
BenchmarkSimpleLRUAdd_int_int-8          4117924               291.1 ns/op            89 B/op          3 allocs/op
BenchmarkSimpleLRUAdd_int_int128-8       3699376               319.2 ns/op           105 B/op          4 allocs/op
BenchmarkSimpleLRUAdd_int_string-8       3667569               328.7 ns/op           105 B/op          4 allocs/op
BenchmarkFreeCacheAdd_int_int-8          5694402               207.0 ns/op             2 B/op          0 allocs/op
BenchmarkFreeCacheAdd_int_int128-8       5943618               206.4 ns/op             1 B/op          0 allocs/op
BenchmarkFreeCacheAdd_int_string-8       4774111               244.2 ns/op            65 B/op          1 allocs/op
BenchmarkMapAdd_int_int-8               12109748               145.6 ns/op             2 B/op          0 allocs/op
BenchmarkMapAdd_int_int128-8            10714936               155.7 ns/op             1 B/op          0 allocs/op
```

Just to give you an idea for 100% memory overcommitment:
Performance increased by ~70% !

```
BenchmarkFreeLRUAdd_int_int-8           17250180                66.62 ns/op            0 B/op          0 allocs/op
BenchmarkFreeLRUAdd_int_int128-8        17454070                67.24 ns/op            0 B/op          0 allocs/op
```

### Getting objects

FreeLRU Get() is ~27% faster than SimpleLRU, but ~2x slower than a straight Go map lookup.

This is with 25% memory overcommitment (default).
```
BenchmarkFreeLRUGet-8                   51072902                23.34 ns/op            0 B/op          0 allocs/op
BenchmarkSimpleLRUGet-8                 38223922                29.60 ns/op            0 B/op          0 allocs/op
BenchmarkFreeCacheGet-8                 30924639                33.38 ns/op            0 B/op          0 allocs/op
BenchmarkMapGet-8                       89974257                12.28 ns/op            0 B/op          0 allocs/op
```

With 100% memory overcommitment, the performance is getting better.
Here, FreeLRU is 70% faster than SimpleLRU and just 37% slower than Go maps.
```
BenchmarkFreeLRUGet-8                   70516069                17.06 ns/op            0 B/op          0 allocs/op
```

### Can we do better ?

Surely yes.
A specialized code without generics, e.g. a `uint32_uint64` code with 100% memory
overcommitment is slightly faster
```
BenchmarkFreeLRUAdd_uint32_uint64-8     24647635                46.85 ns/op            0 B/op          0 allocs/op
```

FreeLRU with the same types (you find the benchmark code in the `lru_test.go`), 100% overcommitment:
```
BenchmarkFreeLRUAdd_uint32_uint64-8     22154940                53.32 ns/op            0 B/op          0 allocs/op
```

Another improvement could be a faster hash algorithm to create the hash from the key.
Also, implementing a different hashtable algorithm that is better on collisions could give a boost.

## Example usage
```
    lru, _ := New[int, uint64](8192, nil, hashInt)
    k := 123
    v := uint64(999)
    lru.Add(k, v)
    v, ok = lru.Get(k)

```
The function `hashInt(int) uint32` will be called to hash the key (of type `int` here).
The FNV1a hash algorithm does a good job (fast and 'random').
Please take a look into `lru_test.go` to find an example FNV1a implementation.

In case you already have a hash that you want to use as key, you need to provide an "identity" function.
Have a look at `BenchmarkFreeLRUAdd_uint32_uint64()` as an example.
