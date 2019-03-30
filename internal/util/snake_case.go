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

package util

import (
	"strings"
)

func isSnakeCaseLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func toSnakeCaseLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}

// SnakeCase converts a string of the form "/[_A-Za-z][_0-9A-Za-z]*/" [0] into snake case. For
// example, it returns "snake_case" for "SnakeCase".
//
// [0]: https://graphql.github.io/graphql-spec/June2018/#Name
func SnakeCase(s string) string {
	sLen := len(s)
	if sLen == 0 {
		return s
	} else if sLen == 1 {
		return strings.ToLower(s)
	}

	var buf StringBuilder
	buf.Grow(sLen)

	var (
		prev = s[0]
		cur  = s[1]
	)

	// Handle the first character.
	buf.WriteByte(toSnakeCaseLower(prev))

	for i := 1; i < sLen-1; i++ {
		var (
			next  = s[i+1]
			lower = toSnakeCaseLower(cur)
		)

		if lower != cur {
			if prev != '_' &&
				(isSnakeCaseLower(prev) || isSnakeCaseLower(next)) {
				buf.WriteByte('_')
			}
		}
		buf.WriteByte(lower)

		prev = cur
		cur = next
	}

	// Handle the last character.
	{
		var lower = toSnakeCaseLower(cur)

		if lower != cur {
			if prev != '_' &&
				(isSnakeCaseLower(prev) /* || unicode.IsLower(next) */) {
				buf.WriteRune('_')
			}
		}
		buf.WriteByte(lower)
	}

	return buf.String()
}
