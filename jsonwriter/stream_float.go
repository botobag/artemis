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
	"encoding/json"
	"math"
	"reflect"
	"strconv"
)

// WriteFloat32 writes a float32.
//
// Implementation mirrored from https://go.googlesource.com/go/+/5fae09b/src/encoding/json/encode.go#546.
func (stream *Stream) WriteFloat32(f float32) {
	if stream.err != nil {
		return
	}

	f64 := float64(f)
	if math.IsInf(f64, 0) || math.IsNaN(f64) {
		stream.err = &json.UnsupportedValueError{
			Value: reflect.ValueOf(f),
			Str:   strconv.FormatFloat(f64, 'g', -1, 32),
		}
		return
	}

	abs := math.Abs(f64)
	fmt := byte('f')
	// Note: Must use float32 comparisons for underlying float32 value to get precise cutoffs right.
	if abs != 0 {
		if float32(abs) < 1e-6 || float32(abs) >= 1e21 {
			fmt = 'e'
		}
	}

	b := strconv.AppendFloat(stream.scratch[:0], f64, fmt, -1, 32)
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}

	stream.write(b)
}

// WriteFloat64 writes a float64.
//
// Implementation mirrored from https://go.googlesource.com/go/+/5fae09b/src/encoding/json/encode.go#546.
func (stream *Stream) WriteFloat64(f float64) {
	if stream.err != nil {
		return
	}

	if math.IsInf(f, 0) || math.IsNaN(f) {
		stream.err = &json.UnsupportedValueError{
			Value: reflect.ValueOf(f),
			Str:   strconv.FormatFloat(f, 'g', -1, 64),
		}
		return
	}

	abs := math.Abs(f)
	fmt := byte('f')
	if abs != 0 {
		if abs < 1e-6 || abs >= 1e21 {
			fmt = 'e'
		}
	}

	b := strconv.AppendFloat(stream.scratch[:0], f, fmt, -1, 64)
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}

	stream.write(b)
}
