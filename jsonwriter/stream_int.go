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
	"strconv"
)

// WriteInt writes an int.
func (stream *Stream) WriteInt(i int) {
	stream.WriteInt64(int64(i))
}

// WriteInt8 writes an int8.
func (stream *Stream) WriteInt8(i int8) {
	stream.WriteInt64(int64(i))
}

// WriteInt16 writes an int16.
func (stream *Stream) WriteInt16(i int16) {
	stream.WriteInt64(int64(i))
}

// WriteInt32 writes an int32.
func (stream *Stream) WriteInt32(i int32) {
	stream.WriteInt64(int64(i))
}

// WriteInt64 writes an int64.
func (stream *Stream) WriteInt64(i int64) {
	stream.write(strconv.AppendInt(stream.scratch[:0], i, 10))
}

// WriteUint writes an int.
func (stream *Stream) WriteUint(i uint) {
	stream.WriteUint64(uint64(i))
}

// WriteUint8 writes an int8.
func (stream *Stream) WriteUint8(i uint8) {
	stream.WriteUint64(uint64(i))
}

// WriteUint16 writes an int16.
func (stream *Stream) WriteUint16(i uint16) {
	stream.WriteUint64(uint64(i))
}

// WriteUint32 writes an int32.
func (stream *Stream) WriteUint32(i uint32) {
	stream.WriteUint64(uint64(i))
}

// WriteUint64 writes an int64.
func (stream *Stream) WriteUint64(i uint64) {
	stream.write(strconv.AppendUint(stream.scratch[:0], i, 10))
}
