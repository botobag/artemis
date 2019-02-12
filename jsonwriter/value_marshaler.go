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

package jsonwriter

import (
	"bytes"
	"encoding/json"
	"reflect"
)

// ValueMarshaler is the interface implemented by types that can marshal themselves to JSON into a stream.
type ValueMarshaler interface {
	MarshalJSONTo(stream *Stream) error
}

// WriteValue writes a value that implements ValueMarshaler.
func (stream *Stream) WriteValue(marshaller ValueMarshaler) {
	if stream.err != nil {
		// Quick return if stream is erroneous.
		return
	}

	// Handle null pointers like encoding/json.marshalerEncoder [0].
	//
	// [0]: https://go.googlesource.com/go/+/5fae09b/src/encoding/json/encode.go#445.
	value := reflect.ValueOf(marshaller)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		stream.WriteNil()
		return
	}

	if err := marshaller.MarshalJSONTo(stream); err != nil {
		// Preserve the previous error.
		if stream.err == nil {
			stream.err = &json.MarshalerError{
				Type: value.Type(),
				Err:  err,
			}
		}
	}
}

// Marshal returns the JSON encoding of v that implements ValueMarshaler. It is useful for
// implements type's MarshalJSON to adapt encoding/json Marshaler API.
func Marshal(v ValueMarshaler) ([]byte, error) {
	// We choose to implement a simplified version of WriteValue instead of calling WriteValue for the
	// following reasons:
	//
	//  1. We don't need to check stream.err at the beginning of marshaling.
	//  2. We don't want the error to be wrapped in a json.MarshalerError. Let encoding/json do this
	//     for us.

	// Handle null pointers like encoding/json.marshalerEncoder [0].
	//
	// [0]: https://go.googlesource.com/go/+/5fae09b/src/encoding/json/encode.go#445.
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return []byte{'n', 'u', 'l', 'l'}, nil
	}

	var (
		buf    bytes.Buffer
		stream = NewStream(&buf)
	)

	if err := v.MarshalJSONTo(stream); err != nil {
		return nil, err
	}

	if err := stream.Flush(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
