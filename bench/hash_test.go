package benchmarks

import (
	"encoding/binary"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/xxh3"
)

func BenchmarkHashInt_MapHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashIntMapHash(i)
	}
}

func BenchmarkHashInt_MapHasher(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashIntMapHasher(i)
	}
}

func BenchmarkHashInt_FNV1A(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashIntFNV1A(i)
	}
}

func BenchmarkHashInt_FNV1AUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashIntFNV1AUnsafe(i)
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

func BenchmarkHashString_MapHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashStringMapHash(testString)
	}
}

func BenchmarkHashString_MapHasher(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = hashStringMapHasher(testString)
	}
}

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
