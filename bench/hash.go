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
	"hash/maphash"
	"unsafe"

	"github.com/cespare/xxhash/v2"
	hackymaphash "github.com/dolthub/maphash"
	"github.com/elastic/go-freelru"
	"github.com/zeebo/xxh3"
)

//go:noescape
func strhash(s string, h uintptr) uintptr

//go:noescape
func inthash(i int, h uintptr) uintptr

// PtrSize is the size of a pointer in bytes - unsafe.Sizeof(uintptr(0)) but as an ideal constant.
// It is also the size of the machine's native word size (4 on 32-bit systems, 8 on 64-bit).
const PtrSize = 4 << (^uintptr(0) >> 63)

const hashRandomBytes = PtrSize / 4 * 64

const (
	// FNV-1a
	offset32 = uint32(2166136261)
	prime32  = uint32(16777619)

	// Init32 is what 32 bits hash values should be initialized with.
	Init32 = offset32
)

// used in asm_{386,amd64,arm64}.s to seed the hash function
var aeskeysched [hashRandomBytes]byte

func init() {
	for i := 0; i < hashRandomBytes; i++ {
		aeskeysched[i] = byte(i + 1)
	}
	aeskeysched[hashRandomBytes-1] = 0xFF
}

var stdMapHash maphash.Hash

func hashIntMapHash(i int) uint32 {
	b := unsafe.Slice((*byte)(unsafe.Pointer(&i)), 8)
	stdMapHash.Reset()
	stdMapHash.Write(b)
	return uint32(stdMapHash.Sum64())
}

func hashStringMapHash(s string) uint32 {
	stdMapHash.WriteString(s)
	return uint32(stdMapHash.Sum64())
}

var hackyIntMapHasher = hackymaphash.NewHasher[int]()

func hashIntMapHasher(i int) uint32 {
	return uint32(hackyIntMapHasher.Hash(i))
}

var hackyStringMapHasher = hackymaphash.NewHasher[string]()

func hashStringMapHasher(s string) uint32 {
	return uint32(hackyStringMapHasher.Hash(s))
}

func hashIntFNV1A(i int) uint32 {
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], uint64(i))
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

func hashIntFNV1AUnsafe(i int) uint32 {
	b := unsafe.Slice((*byte)(unsafe.Pointer(&i)), 8)
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

func hashIntAESENC(i int) uint32 {
	return uint32(inthash(i, 0x1234bacd9876feda))
}

func getHashAESENC[K comparable]() freelru.HashKeyCallback[K] {
	var k K
	switch any(k).(type) {
	case string:
		return any((freelru.HashKeyCallback[string])(hashStringAESENC)).(freelru.HashKeyCallback[K])
	case int:
		return any((freelru.HashKeyCallback[int])(hashIntAESENC)).(freelru.HashKeyCallback[K])
	}
	return nil
}

func hashUInt32(i uint32) uint32 {
	b := [4]byte{}
	binary.BigEndian.PutUint32(b[:], i)
	h := Init32
	h = (h ^ uint32(b[0])) * prime32
	h = (h ^ uint32(b[1])) * prime32
	h = (h ^ uint32(b[2])) * prime32
	h = (h ^ uint32(b[3])) * prime32
	return h
}

// nolint: deadcode, unused
func hashBytes(b []byte) uint32 {
	h := Init32
	for ; len(b) >= 8; b = b[8:] {
		h = (h ^ uint32(b[0])) * prime32
		h = (h ^ uint32(b[1])) * prime32
		h = (h ^ uint32(b[2])) * prime32
		h = (h ^ uint32(b[3])) * prime32
		h = (h ^ uint32(b[4])) * prime32
		h = (h ^ uint32(b[5])) * prime32
		h = (h ^ uint32(b[6])) * prime32
		h = (h ^ uint32(b[7])) * prime32
	}

	if len(b) >= 4 {
		h = (h ^ uint32(b[0])) * prime32
		h = (h ^ uint32(b[1])) * prime32
		h = (h ^ uint32(b[2])) * prime32
		h = (h ^ uint32(b[3])) * prime32
		b = b[4:]
	}

	if len(b) >= 2 {
		h = (h ^ uint32(b[0])) * prime32
		h = (h ^ uint32(b[1])) * prime32
		b = b[2:]
	}

	if len(b) != 0 {
		h = (h ^ uint32(b[0])) * prime32
	}

	return h
}

func hashStringFNV1A(s string) uint32 {
	// ~2x faster than hashBytes([]byte(s)) by avoiding a copy.
	b := *(*[]byte)(unsafe.Pointer(&sliceHeader{s, len(s)}))
	return hashBytes(b)
}

// sliceHeader is similar to reflect.SliceHeader, but it assumes that the layout
// of the first two words is the same as the layout of a string.
type sliceHeader struct {
	s   string
	cap int
}

func hashStringXXHASH(s string) uint32 {
	return uint32(xxhash.Sum64String(s))
}

func hashStringXXH3HASH(s string) uint32 {
	return uint32(xxh3.HashString(s))
}

// go:noescape
func hashStringAESENC(s string) uint32 {
	return uint32(strhash(s, 0x123456789))
}

type int128 struct {
	lo, hi uint64
}
