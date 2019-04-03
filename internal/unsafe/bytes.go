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

// Contents in this file are mostly from https://github.com/m3db/m3x/blob/e98ec32/unsafe/string.go.
// The license is reproduced below.

// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package unsafe

import (
	"reflect"
	"unsafe"
)

// String returns a string backed by a byte slice, it is the caller's responsibility not to mutate
// the bytes while using the string returned.
func String(b []byte) string {
	var s string
	if len(b) == 0 {
		return s
	}

	// We need to declare a real string so internally the compiler knows to use an unsafe.Pointer to
	// keep track of the underlying memory so that once the strings's array pointer is updated with
	// the pointer to the byte slices's underlying bytes, the compiler won't prematurely GC the memory
	// when the byte slice goes out of scope.
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))

	// This makes sure that even if GC relocates the byte slices's underlying memory after this
	// assignment, the corresponding unsafe.Pointer in the internal string struct will be updated
	// accordingly to reflect the memory relocation.
	stringHeader.Data = (*reflect.SliceHeader)(unsafe.Pointer(&b)).Data

	// It is important that we access b after we assign the Data pointer of the byte slice header to
	// the Data pointer of the string header to make sure the bytes don't get GC'ed before the
	// assignment happens.
	l := len(b)
	stringHeader.Len = l

	return s
}
