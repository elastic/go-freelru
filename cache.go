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

package freelru

import "time"

type Cache[K comparable, V any] interface {
	// SetLifetime sets the default lifetime of LRU elements.
	// Lifetime 0 means "forever".
	SetLifetime(lifetime time.Duration)

	// SetOnEvict sets the OnEvict callback function.
	// The onEvict function is called for each evicted lru entry.
	SetOnEvict(onEvict OnEvictCallback[K, V])

	// Len returns the number of elements stored in the cache.
	Len() int

	// AddWithLifetime adds a key:value to the cache with a lifetime.
	// Returns true, true if key was updated and eviction occurred.
	AddWithLifetime(key K, value V, lifetime time.Duration) (evicted bool)

	// Add adds a key:value to the cache.
	// Returns true, true if key was updated and eviction occurred.
	Add(key K, value V) (evicted bool)

	// Get retrieves an element from map under given key.
	Get(key K) (V, bool)

	// Peek retrieves an element from map under given key without updating the
	// "recently used"-ness of that key.
	Peek(key K) (V, bool)

	// Contains checks for the existence of a key, without changing its recent-ness.
	Contains(key K) bool

	// Remove removes an element from the map.
	// The return value indicates whether the key existed or not.
	Remove(key K) bool

	// RemoveOldest removes the oldest entry from the cache.
	RemoveOldest() (key K, value V, removed bool)

	// Keys returns a slice of the keys in the cache, from oldest to newest.
	Keys() []K

	// Purge purges all data (key and value) from the LRU.
	Purge()

	// Metrics returns the metrics of the cache.
	Metrics() Metrics

	// ResetMetrics resets the metrics of the cache and returns the previous state.
	ResetMetrics() Metrics
}
