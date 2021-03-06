/**
 * Copyright (c) 2018, The Artemis Authors.
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

package token

import (
	"unicode/utf8"

	"github.com/botobag/artemis/internal/unsafe"
)

// SourceBody contains contents of a GraphQL document in a byte sequence.
type SourceBody []byte

// RuneAt decodes a rune at given pos. It also returns the number of bytes occupied by the
// rune.
func (body SourceBody) RuneAt(pos uint) (rune, uint) {
	if uint(len(body)) <= pos {
		// Return -1 to indicate an <EOF>.
		return -1, 0
	}

	// Fast path: characters below Runeself are represented as themselves in a single byte.
	c := body[pos]
	if c < utf8.RuneSelf {
		return rune(c), 1
	}

	r, n := utf8.DecodeRune(body[pos:])
	return r, uint(n)
}

// At returns the byte in the source at given position. Return 0 if the given position is out of
// body's range.
func (body SourceBody) At(pos uint) byte {
	if body.Size() <= pos {
		return 0
	}
	return body[pos]
}

// Size returns the body size in bytes.
func (body SourceBody) Size() uint {
	return uint(len(body))
}

// SubStr returns a string that comprises bytes between [start, end) in body.
func (body SourceBody) SubStr(start uint, end uint) string {
	return unsafe.String(body[start:end])
}

// SourceLocationInfo describes a source location for a SourceLocation with source name, line and
// column number.
type SourceLocationInfo struct {
	Name   string
	Line   uint
	Column uint
}

// Source represent a GraphQL source text.
//
// It can be configure with optional name, line offset and column offset (via SourceName,
// SourceLineOffset, and SourceColumnOffset, respectively) on creation. These properties are useful
// for clients who store GraphQL documents in source files. For example, if the GraphQL input starts
// at line 40 in a file named Foo.graphql, it might be useful to set name to be "Foo.graphql" with
// line and column offset set to 40 and 0, respecitievly. Note that the offsets are both 0-indexed
// and are both 0 if they're not provided (which also means no offset).
type Source struct {
	body         SourceBody
	name         string
	lineOffset   uint
	columnOffset uint
}

// SourceOption configures a Source instance.
type SourceOption func(*Source)

// SourceName specifies name of a Source to be shown in printing a source location.
func SourceName(name string) SourceOption {
	return func(source *Source) {
		source.name = name
	}
}

// SourceLineOffset specifies line offset to be added to a source location. See comments for Source
// for details.
func SourceLineOffset(offset uint) SourceOption {
	return func(source *Source) {
		source.lineOffset = offset
	}
}

// SourceColumnOffset specifies column offset to be added to a source location. See comments for
// Source for details.
func SourceColumnOffset(offset uint) SourceOption {
	return func(source *Source) {
		source.columnOffset = offset
	}
}

// NewSource initializes a Source instance from given string and options.
func NewSource(s string, opts ...SourceOption) *Source {
	// Take internal buffer that backed the string. This avoids copy. This should be considered safe
	// in our use case since SourceBody is read-only.
	return NewSourceFromBytes(unsafe.Bytes(s), opts...)
}

// NewSourceFromBytes initializes a Source instance from given byte slice and options.
func NewSourceFromBytes(b []byte, opts ...SourceOption) *Source {
	source := &Source{
		body: SourceBody(b),
		name: "GraphQL request",
	}

	// Apply options.
	for _, opt := range opts {
		opt(source)
	}

	return source
}

// Body returns source.config.Body.
func (source *Source) Body() SourceBody {
	return source.body
}

// Name returns source.config.Name.
func (source *Source) Name() string {
	return source.name
}

// LineOffset returns source.config.LineOffset.
func (source *Source) LineOffset() uint {
	return source.lineOffset
}

// ColumnOffset returns source.config.ColumnOffset.
func (source *Source) ColumnOffset() uint {
	return source.columnOffset
}

// LocationFromPos returns a SourceLocation that represent the location for given position in the
// body.
func (source *Source) LocationFromPos(bytePos uint) SourceLocation {
	if bytePos > source.Body().Size() {
		panic("illegal byte position value")
	}
	return SourceLocation(bytePos + 1)
}

// PosFromLocation is a reverse operation of LocationFromPos. It converts the given SourceLocation
// to the byte position in the source which is a 0-based offset relative to the beginning of the
// source body.
func (source *Source) PosFromLocation(location SourceLocation) uint {
	if !location.IsValid() || uint(location) > (source.Body().Size()+1) {
		panic("illegal location value")
	}
	return uint(location) - 1
}

// LocationInfoOf computes and returns a SourceLocationInfo for a given SourceLocation.
func (source *Source) LocationInfoOf(loc SourceLocation) SourceLocationInfo {
	// TODO: Cache table of line offsets for a Source for the first time this is called. #5

	// Handle invalid SourceLocation (NoSourceLocation). This may happen when querying location for
	// special token like SOF which inherently has no source location.
	if !loc.IsValid() {
		return SourceLocationInfo{
			Name: source.Name(),
		}
	}

	var (
		line     uint = 1
		column   uint = 1
		position      = uint(loc) - 1
	)

	body := source.Body()
	bodySize := body.Size()
	if position >= bodySize {
		position = bodySize
	}

	var i uint
	for i < position {
		switch body[i] {
		case '\r':
			if (i+1) < bodySize && body[i+1] == '\n' {
				// An "\r\n" was encountered and we're at "\r". Both graphql-js and graphql-go consider the
				// position of "\r" at the same line. So don't advance line (and column).
				i++

				// Now consume "\n". Here is the special case: if position of "\n" is requested, it is in
				// the next line with column number as 0. Otherwise (i.e., the requesting position is not
				// "\n"), we process the "\n" as normal case.
				if i == position {
					line++
					column = 0
				}
			} else {
				line++
				column = 1
				i++
			}

		case '\n':
			line++
			column = 1
			i++

		default:
			column++
			i++
		}
	}

	return SourceLocationInfo{
		Name:   source.Name(),
		Line:   source.LineOffset() + line,
		Column: source.ColumnOffset() + column,
	}
}
