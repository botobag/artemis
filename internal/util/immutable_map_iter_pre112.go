// Only build for Go pre-1.12 where reflect.MapIter is not available.
//+build !go1.12

/**
 * Copyright (c) 2019, The Artemis Authors.
 *
 * Permission to use, copy, modify, and/or distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package util

import (
	"reflect"
)

// This files provides a "shim" layer to MapIter feature which is only available in Go 1.12 and
// later. On first iteration (i.e., first call to Next), map keys are populated into a slice with
// reflect.Value.MapKeys and subsequent iterations loop over the slice. Value in current iteration
// is obtained via reflect.Value.MapIndex. There're 2 caveats:
//
//   1. Without native runtime support, this won't work efficiently as reflect.MapIter;
//   2. This does *NOT* follow the exact same iteration semantics as a range statement, more
//      specifically when the underlying map is modified during iteration. This makes the iterator
//      only useful when iterating an "immutable" map.
//      - If the specified key is removed from the underlying map after first call to Next, you
//        may get the zero value from iter.Value. (While reflect.MapIter would )
//      - New entry added to the map after first call to Next won't be visible to the iterator.

// ImmutableMapIter provides iterator to loop over a map. The map is assumed to remain unmodified
// during iteration.
//
// Call Next to advance the iterator, and Key/Value to access each entry. Next returns false when
// the iterator is exhausted.
//
// Example:
//
//	iter := NewMapIter(m)
//	for iter.Next() {
//		k := iter.Key()
//		v := iter.Value()
//		...
//	}
//
type ImmutableMapIter struct {
	// Value of the map
	m reflect.Value
	// Keys of the map; It is lazily initialized in first Next call.
	keys []reflect.Value
	// Index of keys of the iterator's current map entry.
	i int
}

// NewImmutableMapIter creates a MapIter to loop over the given m. It panics if m's Kind is not Map.
func NewImmutableMapIter(m interface{}) *ImmutableMapIter {
	v := reflect.ValueOf(m)
	// Make sure v is a map. Mimic reflect.mustBe(Map).
	if v.Kind() != reflect.Map {
		panic(&reflect.ValueError{
			Method: "github.com/botobag/artemis/internal/util.NewMapIter",
			Kind:   v.Kind(),
		})
	}

	return &ImmutableMapIter{
		m: v,
	}
}

// The following implements the same set of interfaces provided by reflect.MapIter.

// Key returns the key of the iterator's current map entry.
func (it *ImmutableMapIter) Key() reflect.Value {
	return it.keys[it.i]
}

// Value returns the value of the iterator's current map entry.
func (it *ImmutableMapIter) Value() reflect.Value {
	return it.m.MapIndex(it.Key())
}

// Next advances the map iterator and reports whether there is another entry. It returns false when
// the iterator is exhausted; subsequent calls to Key, Value, or Next will panic.
func (it *ImmutableMapIter) Next() bool {
	if it.keys == nil {
		it.keys = it.m.MapKeys()
	} else {
		// Advance index.
		it.i++
	}
	return it.i < len(it.keys)
}
