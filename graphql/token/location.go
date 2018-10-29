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

// SourceLocation encodes a position in source file. It lives in the context of Source. Its value is
// an 1-indexed offset relative to the beginning of source measured in bytes. Given a SourceLocation
// value loc and the Source s, you can convert it into larger representation SourceLocationInfo by
// calling s.LocationInfoOf(loc).
type SourceLocation uint

// NoSourceLocation is a special SourceLocation that doesn't exists in any source. Method that deals
// with SourceLocation must take special care to handle this value.
const NoSourceLocation SourceLocation = 0

// IsValid return true if the SourceLocation is valid.
func (location SourceLocation) IsValid() bool {
	return location != NoSourceLocation
}

// WithOffset returns a source location with the specified offset from this location.
func (location SourceLocation) WithOffset(offset int) SourceLocation {
	return SourceLocation(int(location) + offset)
}

// SourceRange is a tuple used to represent a source range.
type SourceRange struct {
	// [Begin, End)
	Begin SourceLocation
	End   SourceLocation
}

// IsValid return true if both loactions in the range are valid.
func (r SourceRange) IsValid() bool {
	return r.Begin.IsValid() && r.End.IsValid()
}
