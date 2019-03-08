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

package executor

import (
	"reflect"

	"github.com/botobag/artemis/internal/util"
	"github.com/botobag/artemis/iterator"
)

// An Iterable defines iteration behavior. It is recognized by executor specially when it is
// presented to a field of a List type.
type Iterable interface {
	// Iterator returns iterator to loop over its values.
	Iterator() Iterator
}

// SizedIterable provides hint about size of iterable.
type SizedIterable interface {
	Iterable

	// Size provides hint about number of values in the sequence.
	Size() int
}

// Iterator defines a way to access values in an Iterable.
type Iterator interface {
	// Next returns the next value in iteration. It follows the semantics defined by iterator package
	// [0] which returns:
	//
	//  - (value, nil): return the next value in sequence.
	//  - (<ignored>, iterator.Done): the iterator is past the end of the iterated sequence.
	//  - (<ignored>, <error>): there's an error occurred when fetching the next value in sequence.
	//
	// [0]: github.com/botobag/artemis/iterator
	Next() (interface{}, error)
}

//===----------------------------------------------------------------------------------------====//
// MapKeysIterable
//===----------------------------------------------------------------------------------------====//

// MapKeysIterable wraps a Go map into an Iterable and provides an iterator to loop over keys in the
// map. Note that the given map should not be modified during iteration.
type MapKeysIterable struct {
	// The map to be iterated; Must be a Go map.
	m interface{}
}

// NewMapKeysIterable creates a MapKeysIterable. m must be a Go map.
func NewMapKeysIterable(m interface{}) *MapKeysIterable {
	return &MapKeysIterable{m}
}

// Iterator implements Iterable. It returns iterator for iterating map keys.
func (iterable *MapKeysIterable) Iterator() Iterator {
	return MapKeysIterator{util.NewImmutableMapIter(iterable.m)}
}

// Size implements SizedIterable. It returns the number of entries in the map.
func (iterable *MapKeysIterable) Size() int {
	return reflect.ValueOf(iterable.m).Len()
}

// MapKeysIterator implements Iterator to loop over the keys in a map.
type MapKeysIterator struct {
	iter *util.ImmutableMapIter
}

// Next implements Iterator.
func (iter MapKeysIterator) Next() (interface{}, error) {
	mapIter := iter.iter
	if !mapIter.Next() {
		return nil, iterator.Done
	}
	return mapIter.Key().Interface(), nil
}

//===----------------------------------------------------------------------------------------====//
// MapValuesIterable
//===----------------------------------------------------------------------------------------====//

// MapValuesIterable wraps a Go map into an Iterable and provides an iterator to loop over the
// values in the map. Note that the given map should not be modified during iteration.
type MapValuesIterable struct {
	// The map to be iterated; Must be a Go map.
	m interface{}
}

// NewMapValuesIterable creates a MapValuesIterable. m must be a Go map.
func NewMapValuesIterable(m interface{}) *MapValuesIterable {
	return &MapValuesIterable{m}
}

// Iterator implements Iterable. It returns iterator for iterating map values.
func (iterable *MapValuesIterable) Iterator() Iterator {
	return MapValuesIterator{util.NewImmutableMapIter(iterable.m)}
}

// Size implements SizedIterable. It returns the number of entries in the map.
func (iterable *MapValuesIterable) Size() int {
	return reflect.ValueOf(iterable.m).Len()
}

// MapValuesIterator implements Iterator to loop over the values in a map.
type MapValuesIterator struct {
	iter *util.ImmutableMapIter
}

// Next implements Iterator.
func (iter MapValuesIterator) Next() (interface{}, error) {
	mapIter := iter.iter
	if !mapIter.Next() {
		return nil, iterator.Done
	}
	return mapIter.Value().Interface(), nil
}
