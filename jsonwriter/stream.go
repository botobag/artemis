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
	"io"
	"reflect"
)

const initialStreamBufSize = 512

// Stream provides functions for writing JSON encoding. Unlike encoding/json, the writes are
// directly sent to to the output via io.Writer.
type Stream struct {
	// Output stream
	w io.Writer

	// Buffer that sits in front of write to w; Its capacity is initialized to 512 bytes and may grow
	// indefinitely if there're many write{One,Two,...}Byte{s} calls. This is intended to make
	// write{One,Two,...}Byte{s} fast which is critical in our micro-benchmark
	// (see graphql/executor/result_marshaler_benchmark_test.go).
	buf []byte

	// Buffer for Write{Int64,Uint64,Float32,Float64}
	scratch [64]byte

	// Go's encoding/json.Encoder for encoding values that cannot be proccessed by this writer.
	fallbackEncoder *json.Encoder

	// Error occurred during writing
	err error
}

// NewStream creates a stream for writing data in JSON encoding.
func NewStream(w io.Writer) *Stream {
	return &Stream{
		w:   w,
		buf: make([]byte, 0, initialStreamBufSize),
	}
}

// Error returns error occurred during use of the stream.
func (stream *Stream) Error() error {
	return stream.err
}

// write is the lowest level that performs writes. It writes the contents given in b into w.
func (stream *Stream) write(b []byte) {
	// Discard writes if error already occurred in prior to the write.
	if stream.err != nil {
		return
	}

	buf := stream.buf
	bufSize := len(buf)
	if bufSize+len(b) < initialStreamBufSize {
		buf = buf[:bufSize+len(b)]
		stream.buf = buf
		copy(buf[bufSize:], b)
		return
	}

	if bufSize > 0 {
		_, err := stream.w.Write(buf)
		// Reset buf.
		stream.buf = buf[:0]
		if err != nil {
			stream.err = err
			return
		}
	}

	if len(b) > 0 {
		if _, err := stream.w.Write(b); err != nil {
			stream.err = err
			return
		}
	}
}

// Flush writes any buffered data to the underlying io.Writer.
func (stream *Stream) Flush() error {
	if stream.err != nil {
		return stream.err
	}

	buf := stream.buf
	if len(buf) > 0 {
		_, err := stream.w.Write(buf)
		// Reset buf.
		stream.buf = buf[:0]
		if err != nil {
			stream.err = err
			return err
		}
	}

	return nil
}

func (stream *Stream) writeOneByte(b byte) {
	stream.buf = append(stream.buf, b)
}

func (stream *Stream) writeTwoBytes(b1 byte, b2 byte) {
	stream.buf = append(stream.buf, b1, b2)
}

func (stream *Stream) writeThreeBytes(b1 byte, b2 byte, b3 byte) {
	stream.buf = append(stream.buf, b1, b2, b3)
}

func (stream *Stream) writeFourBytes(b1 byte, b2 byte, b3 byte, b4 byte) {
	stream.buf = append(stream.buf, b1, b2, b3, b4)
}

func (stream *Stream) writeFiveBytes(b1 byte, b2 byte, b3 byte, b4 byte, b5 byte) {
	stream.buf = append(stream.buf, b1, b2, b3, b4, b5)
}

func (stream *Stream) writeSixBytes(b1 byte, b2 byte, b3 byte, b4 byte, b5 byte, b6 byte) {
	stream.buf = append(stream.buf, b1, b2, b3, b4, b5, b6)
}

// WriteRawString writes raw string into output.
func (stream *Stream) WriteRawString(s string) {
	stream.write([]byte(s))
}

// WriteMore writes a ",".
func (stream *Stream) WriteMore() {
	stream.writeOneByte(',')
}

// WriteArrayStart writes a "[".
func (stream *Stream) WriteArrayStart() {
	stream.writeOneByte('[')
}

// WriteArrayEnd writes a "]".
func (stream *Stream) WriteArrayEnd() {
	stream.writeOneByte(']')
}

// WriteEmptyArray writes "[]".
func (stream *Stream) WriteEmptyArray() {
	stream.writeTwoBytes('[', ']')
}

// WriteObjectStart writes a "{".
func (stream *Stream) WriteObjectStart() {
	stream.writeOneByte('{')
}

// WriteObjectField writes a "field:".
func (stream *Stream) WriteObjectField(field string) {
	stream.WriteString(field)
	stream.writeOneByte(':')
}

// WriteObjectEnd writes a "}".
func (stream *Stream) WriteObjectEnd() {
	stream.writeOneByte('}')
}

// WriteEmptyObject writes "{}".
func (stream *Stream) WriteEmptyObject() {
	stream.writeTwoBytes('{', '}')
}

// WriteBool encodes a boolean value.
func (stream *Stream) WriteBool(b bool) {
	if b {
		stream.writeFourBytes('t', 'r', 'u', 'e')
	} else {
		stream.writeFiveBytes('f', 'a', 'l', 's', 'e')
	}
}

// WriteNil writes "null".
func (stream *Stream) WriteNil() {
	stream.writeFourBytes('n', 'u', 'l', 'l')
}

// streamWriter wraps a Stream into an io.Writer object.
type streamWriter struct {
	stream *Stream
}

func (writer streamWriter) Write(p []byte) (n int, err error) {
	stream := writer.stream
	stream.write(p)
	err = stream.err
	if err == nil {
		n = len(p)
	}
	return
}

var jsonMarshalerType = reflect.TypeOf(new(json.Marshaler)).Elem()

// WriteInterface writes an interface value using encoding/json.
func (stream *Stream) WriteInterface(v interface{}) {
	if stream.err != nil {
		return
	}

	// Fast path with type switch
	switch v := v.(type) {
	// Bool
	case bool:
		stream.WriteBool(v)
		return
	case *bool:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteBool(*v)
		}

	// String
	case string:
		stream.WriteString(v)
	case *string:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteString(*v)
		}

	// Integer
	case int:
		stream.WriteInt(v)
	case int8:
		stream.WriteInt8(v)
	case int16:
		stream.WriteInt16(v)
	case int32:
		stream.WriteInt32(v)
	case int64:
		stream.WriteInt64(v)
	case uint:
		stream.WriteUint(v)
	case uint8:
		stream.WriteUint8(v)
	case uint16:
		stream.WriteUint16(v)
	case uint32:
		stream.WriteUint32(v)
	case uint64:
		stream.WriteUint64(v)
	case *int:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteInt(*v)
		}
	case *int8:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteInt8(*v)
		}
	case *int16:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteInt16(*v)
		}
	case *int32:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteInt32(*v)
		}
	case *int64:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteInt64(*v)
		}
	case *uint:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteUint(*v)
		}
	case *uint8:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteUint8(*v)
		}
	case *uint16:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteUint16(*v)
		}
	case *uint32:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteUint32(*v)
		}
	case *uint64:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteUint64(*v)
		}

	// Floating point
	case float32:
		stream.WriteFloat32(v)
	case float64:
		stream.WriteFloat64(v)
	case *float32:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteFloat32(*v)
		}
	case *float64:
		if v == nil {
			stream.WriteNil()
		} else {
			stream.WriteFloat64(*v)
		}

	case ValueMarshaler:
		stream.WriteValue(v)

	case nil:
		stream.WriteNil()

	default:
		// Fast path using reflection to inspect value type
		value := reflect.ValueOf(v)

		if value.Type().Implements(jsonMarshalerType) {
			// If value implements json.Marshaler, go to fallback path and let encoding/json handle it.
			stream.writeInterfaceFallback(v)
			return
		}

		switch value.Kind() {
		case reflect.Invalid:
			stream.WriteNil()

		case reflect.Bool:
			stream.WriteBool(value.Bool())

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			stream.WriteInt64(value.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			stream.WriteUint64(value.Uint())

		case reflect.Float32:
			stream.WriteFloat32(float32(value.Float()))
		case reflect.Float64:
			stream.WriteFloat64(value.Float())

		case reflect.String:
			stream.WriteString(value.String())

		case reflect.Ptr:
			elemValue := value.Elem()
			if !elemValue.IsValid() {
				// A nil pointer
				stream.WriteNil()
			} else {
				stream.WriteInterface(elemValue.Interface())
			}

		default:
			// Fallback to encoding/json to encode value.
			stream.writeInterfaceFallback(v)
		}
	}
}

// writeInterfaceFallback is the fallback for WriteInterface which encodes the value using
// encoding/json.
func (stream *Stream) writeInterfaceFallback(v interface{}) {
	encoder := stream.fallbackEncoder
	if encoder == nil {
		encoder = json.NewEncoder(streamWriter{stream})
		stream.fallbackEncoder = encoder
	}

	if err := encoder.Encode(v); err != nil {
		if stream.err == nil {
			stream.err = err
		}
	}
}
