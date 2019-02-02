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

// Package iterator documents the guidelines for using iterator pattern in Artemis. The pattern
// draws significant inspiration from the Iterator Guidelines established for Google Cloud Client
// Libraries for Go [0].
//
// Since Go doesn't have generics, there's no way to provide an "iterator" foundation as seen in
// Java [1]. It might be possible to use reflection and interface{} to build one but it needs to pay
// reflection tax and is definitely not type-safe. Therefore, you're unable to find concrete
// interfaces or structs to define an iterator but examples that show how an iterator look like in
// Artemis.
//
// First, an "iterable" resource provides a method named Iterator which returns an iterator over its
// elements. For example,
//
//	// IntVector provides iterator support for an int array.
//	type IntVector struct {
//		v []int
//	}
//
//	// Iterator returns an iterator over the int's in the vector.
//	func (vector IntVector) Iterator() IntVectorIterator {
//		...
//	}
//
// Or when appropriated, using element name (in plural) is prefer. For example,
//
//	type Shelf struct {
//		books []*Book
//	}
//
//	// Books returns an iterator over the books in the shelf.
//	func (shelf *Shelf) Books() BookIterator {
//		...
//	}
//
// The result iterator will have just one method Next for iterating over individual elements. Take
// the shelf example above, the Next method in BookIterator will look like,
//
//	type BookIterator struct {
//		...
//	}
//
//	// Next returns the next book in the iteration. It returns an error iterator.Done to indicate
//	// that there's no more element.
//	func (iter *BookIterator) Next() (*Book, error) {
//		...
//	}
//
// Now, let's show how the BookIterator is used in code,
//
//	iter := shelf.Books()
//	for {
//		book, err := it.Next()
//		if err == iterator.Done {
//			break
//		} else if err != nil {
//			handleError(err)
//		}
//		process(book)
//	}
//
// [0]: https://github.com/googleapis/google-cloud-go/wiki/Iterator-Guidelines
// [1]: https://docs.oracle.com/javase/8/docs/api/java/util/Iterator.html
package iterator
